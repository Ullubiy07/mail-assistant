package imap

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"log/slog"
	"mail-assistant/internal/config"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	imapmail "github.com/emersion/go-message/mail"

	"mail-assistant/internal/mail"
)

type ConnectMethod = string

const (
	XOAUTH2 ConnectMethod = "XOAUTH2"
	PLAIN   ConnectMethod = "PLAIN"
)

var (
	reText = regexp.MustCompile(`https?://\S+|\[\s*\d+\s*\]|\?utm_[a-z_]+=[^&\s]+&?`)
	reHtml = regexp.MustCompile(`(?s)<(style|script)[^>]*>.*?</(style|script)>|<[^>]*>`)
)

type Creds struct {
	address  string
	email    string
	password string
	token    string
}

type Client struct {
	client *imapclient.Client
	cfg    *config.IMAP

	creds  *Creds
	method ConnectMethod
}

type clientXOAUTH2 struct {
	email string
	token string
}

func New(cfg *config.IMAP, method ConnectMethod, address, email, password, token string) Client {
	return Client{nil, cfg, &Creds{
		address:  address,
		email:    email,
		password: password,
		token:    token,
	}, method}
}

func (c *Client) connect(ctx context.Context) error {
	switch c.method {
	case XOAUTH2:
		return c.connectByXOAUTH2(ctx)
	case PLAIN:
		return c.connectByPassword(ctx)
	}
	return fmt.Errorf("unsupported connection method")
}

func (c *Client) connectByPassword(ctx context.Context) error {
	cl, err := imapclient.DialTLS(c.creds.address, &imapclient.Options{
		Dialer: &net.Dialer{
			Timeout: time.Duration(c.cfg.DialTimeout) * time.Second,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to dial IMAP server: %w", err)
	}
	if err := cl.Login(c.creds.email, c.creds.password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	c.client = cl
	return nil
}

func (c *Client) connectByXOAUTH2(ctx context.Context) error {
	cl, err := imapclient.DialTLS(c.creds.address, &imapclient.Options{
		Dialer: &net.Dialer{
			Timeout: time.Duration(c.cfg.DialTimeout) * time.Second,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect IMAP server: %w", err)
	}
	authClient := &clientXOAUTH2{
		email: c.creds.email,
		token: c.creds.token,
	}
	if err := cl.Authenticate(authClient); err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}
	c.client = cl
	return nil
}

func (c Client) close() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

func (c *clientXOAUTH2) Start() (mech string, ir []byte, err error) {
	str := fmt.Sprintf("user=%s\001auth=Bearer %s\001\001", c.email, c.token)
	ir = []byte(str)
	return "XOAUTH2", ir, nil
}

func (c *clientXOAUTH2) Next(challenge []byte) (response []byte, err error) {
	return nil, nil
}

// for development
func (c Client) AuthMechanisms() ([]string, error) {
	cl, err := imapclient.DialTLS(c.creds.address, &imapclient.Options{
		Dialer: &net.Dialer{
			Timeout: time.Duration(c.cfg.DialTimeout) * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect IMAP server: %w", err)
	}
	cap, err := cl.Capability().Wait()
	if err != nil {
		return nil, fmt.Errorf("capability command failed: %w", err)
	}
	res := cap.AuthMechanisms()
	return res, nil
}

func (c Client) GetFolders(ctx context.Context) ([]string, error) {
	if err := c.connect(ctx); err != nil {
		return nil, nil
	}
	defer c.close()

	resultCh := make(chan []string, 1)
	errCh := make(chan error, 1)

	go func() {
		cmd := c.client.List("", "*", nil)
		defer cmd.Close()

		data, err := cmd.Collect()
		if err != nil {
			errCh <- err
			return
		}
		result := make([]string, 0, len(data))
		for _, item := range data {
			result = append(result, item.Mailbox)
		}
		resultCh <- result
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case folders := <-resultCh:
		return folders, nil
	case err := <-errCh:
		return nil, err
	}
}

func (c Client) GetNewLetters(ctx context.Context, folder string, uid uint32) ([]mail.Letter, error) {
	var letters []mail.Letter

	messages, err := c.fetchMessages(ctx, folder, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages from %s: %w", folder, err)
	}

	var extractErr error
	for _, msg := range messages {
		// if len(msg.Envelope.From) != 0 &&
		// 	msg.Envelope.From[0].Mailbox == "noreply" ||
		// 	msg.Envelope.From[0].Mailbox == "devnull" {
		// 	continue
		// }
		select {
		case <-ctx.Done():
			return letters, extractErr
		default:
		}

		body, err := getMessageBody(msg)
		if err != nil {
			extractErr = err
			continue
		}
		if body == "" {
			continue
		}

		from := mail.Address{}
		if len(msg.Envelope.From) > 0 {
			from = mail.Address{
				Name:    msg.Envelope.From[0].Name,
				Mailbox: msg.Envelope.From[0].Mailbox,
				Host:    msg.Envelope.From[0].Host,
			}
		}

		letters = append(letters, mail.Letter{
			Envelope: mail.Envelope{
				Date:    msg.Envelope.Date,
				Subject: msg.Envelope.Subject,
				From:    from,
				UID:     uint32(msg.UID),
			},
			Body: body,
		})
	}

	if extractErr != nil {
		slog.WarnContext(ctx, "partial failure during extraction",
			"provider", "IMAP",
			"messages", len(messages),
			"extracted", len(letters),
			"folder", folder,
			"err", extractErr)
	}

	return letters, extractErr
}

// fetchMessages returns IMAP messages from the specified folder, where letter.uid > uid
func (c Client) fetchMessages(ctx context.Context, folder string, uid uint32) ([]*imapclient.FetchMessageBuffer, error) {
	if err := c.connect(ctx); err != nil {
		return nil, err
	}
	defer c.close()

	resultCh := make(chan []*imapclient.FetchMessageBuffer, 1)
	errCh := make(chan error, 1)

	go func() {
		mailbox, err := c.client.Select(folder, nil).Wait()
		if err != nil {
			errCh <- err
			return
		}
		if mailbox.NumMessages == 0 {
			resultCh <- nil
			return
		}

		uidSet := imap.UIDSet{}
		uidSet.AddRange(imap.UID(uid+1), imap.UID(mailbox.NumMessages))

		messages, err := c.client.Fetch(uidSet, &imap.FetchOptions{
			Envelope:    true,
			UID:         true,
			BodySection: []*imap.FetchItemBodySection{{}},
		}).Collect()

		if err != nil {
			errCh <- err
			return
		}
		resultCh <- messages
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case messages := <-resultCh:
		return messages, nil
	case err := <-errCh:
		return nil, err
	}
}

// getMessageBody extracts and returns text/plain data from an IMAP message
func getMessageBody(message *imapclient.FetchMessageBuffer) (string, error) {
	body := message.FindBodySection(&imap.FetchItemBodySection{})
	mr, err := imapmail.CreateReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	var htmlText string

	for p, err := mr.NextPart(); err != io.EOF; p, err = mr.NextPart() {
		dataType := p.Header.Get("Content-Type")

		switch {
		case strings.HasPrefix(dataType, "text/plain"):
			body, _ := io.ReadAll(p.Body)
			return cleanPlainText(string(body)), nil
		case strings.HasPrefix(dataType, "text/html"):
			body, _ := io.ReadAll(p.Body)
			htmlText = htmlToText(string(body))
		}
	}
	return htmlText, nil
}

// cleanPlainText clears raw string of unnecessary information
func cleanPlainText(raw string) string {
	text := reText.ReplaceAllString(raw, " ")
	text = removeNotPrintable(text)
	text = strings.Join(strings.Fields(text), " ")
	return text
}

func removeNotPrintable(text string) string {
	var builder strings.Builder
	for _, char := range text {
		if unicode.IsControl(char) {
			continue
		}
		if unicode.IsLetter(char) ||
			unicode.IsDigit(char) ||
			unicode.IsSpace(char) ||
			unicode.IsPunct(char) {

			builder.WriteRune(char)
		}
	}
	return builder.String()
}

// htmlToText converts HTML to plain text
func htmlToText(htmlText string) string {
	text := reHtml.ReplaceAllString(htmlText, " ")
	text = html.UnescapeString(text)
	text = cleanPlainText(text)
	return text
}
