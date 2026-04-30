package client

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"math"
	"regexp"
	"strings"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"

	"mail-assistant/internal/model"
)

var (
	reText = regexp.MustCompile(`https?://\S+|\[\s*\d+\s*\]|\?utm_[a-z_]+=[^&\s]+&?`)
	reHtml = regexp.MustCompile(`(?s)<(style|script)[^>]*>.*?</(style|script)>|<[^>]*>`)
)

type Client struct {
	client *imapclient.Client
}

type clientXOAUTH2 struct {
	email string
	token string
}

func New() Client {
	return Client{}
}

func (c *clientXOAUTH2) Start() (mech string, ir []byte, err error) {
	str := fmt.Sprintf("user=%s\001auth=Bearer %s\001\001", c.email, c.token)
	ir = []byte(str)
	return "XOAUTH2", ir, nil
}

func (c *clientXOAUTH2) Next(challenge []byte) (response []byte, err error) {
	return nil, nil
}

// Connect connects to the IMAP server using a password
func (c *Client) Connect(address, email, password string) error {
	cl, err := imapclient.DialTLS(address, nil)
	if err != nil {
		return fmt.Errorf("failed to dial IMAP server: %v", err)
	}
	if err := cl.Login(email, password).Wait(); err != nil {
		return fmt.Errorf("failed to login: %v", err)
	}
	c.client = cl
	return nil
}

// ConnectByXOAUTH2 connects to the IMAP server by XOAUTH2
func (c *Client) ConnectByXOAUTH2(address, email, token string) error {
	cl, err := imapclient.DialTLS(address, nil)
	if err != nil {
		return fmt.Errorf("failed to dial IMAP server: %v", err)
	}
	authClient := &clientXOAUTH2{
		email: email,
		token: token,
	}
	if err := cl.Authenticate(authClient); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}
	c.client = cl
	return nil
}

// Close closes the connection to the IMAP server
func (c *Client) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close IMAP server connection: %v", err)
	}
	return nil
}

// GetLetters returns all new letters from the specified folder, where letter.uid > uid
//
// Valid folders: Drafts, INBOX, Outbox, Sent, Spam, Trash.
func (c Client) GetNewLetters(folder string, uid uint32) ([]model.Letter, error) {
	var letters []model.Letter

	messages, err := c.fetchMessages(folder, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages from %s", folder)
	}

	var extractErr error
	for _, msg := range messages {
		body, err := getMessageBody(msg)
		if err != nil {
			extractErr = err
			continue
		}
		if body != "" {
			letters = append(letters, model.Letter{
				Envelope: model.Envelope{
					Date:    msg.Envelope.Date,
					Subject: msg.Envelope.Subject,
					From:    msg.Envelope.From,
					UID:     uint32(msg.UID),
				},
				Body: body,
			})
		}
	}
	return letters, extractErr
}

// fetchMessages returns IMAP messages from the specified folder, where letter.uid > uid
func (c Client) fetchMessages(folder string, uid uint32) ([]*imapclient.FetchMessageBuffer, error) {
	mailbox, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return nil, err
	}
	if mailbox.NumMessages == 0 {
		return nil, nil
	}

	options := &imap.FetchOptions{
		Envelope:    true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}

	uidSet := imap.UIDSet{}
	uidSet.AddRange(imap.UID(uid+1), math.MaxUint32)
	res, err := c.client.UIDSearch(&imap.SearchCriteria{
		UID: []imap.UIDSet{uidSet},
	}, nil).Wait()

	if err != nil {
		return nil, err
	}

	messages, err := c.client.Fetch(res.All, options).Collect()
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// getMessageBody extracts and returns text/plain data from an IMAP message
func getMessageBody(message *imapclient.FetchMessageBuffer) (string, error) {
	body := message.FindBodySection(&imap.FetchItemBodySection{})
	mr, err := mail.CreateReader(bytes.NewReader(body))
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
