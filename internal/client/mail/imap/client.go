package imap

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"mail-assistant/internal/config"
	"net"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	imapmail "github.com/emersion/go-message/mail"

	"mail-assistant/internal/client/mail"
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

type Factory struct {
	config config.IMAP
}

type Client struct {
	config config.IMAP
	auth   mail.Auth
}

type clientXOAUTH2 struct {
	email string
	token string
}

func New(config config.IMAP) Factory {
	return Factory{config: config}
}

func (f Factory) NewFetcher(auth mail.Auth) mail.Fetcher {
	return Client{
		config: f.config,
		auth:   auth,
	}
}

func (c Client) connect(ctx context.Context) (*imapclient.Client, error) {
	deadline := time.Now().Add(time.Duration(c.config.DialTimeout) * time.Second)
	dl, ok := ctx.Deadline()
	if ok {
		deadline = dl
	}
	cl, err := imapclient.DialTLS(c.auth.Address, &imapclient.Options{
		Dialer: &net.Dialer{
			Deadline: deadline,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dial IMAP server: %w", err)
	}
	return cl, nil
}

func (c Client) authenticate(conn *imapclient.Client) error {
	switch c.auth.Method {
	case XOAUTH2:
		return c.authenticateByXOAUTH2(conn)
	case PLAIN:
		return c.authenticateByPassword(conn)
	}
	return fmt.Errorf("unsupported connection method")
}

func (c Client) authenticateByPassword(conn *imapclient.Client) error {
	if err := conn.Login(c.auth.Email, c.auth.Password).Wait(); err != nil {
		return fmt.Errorf("login attempt: %w", err)
	}
	return nil
}

func (c Client) authenticateByXOAUTH2(conn *imapclient.Client) error {
	authClient := &clientXOAUTH2{
		email: c.auth.Email,
		token: c.auth.Token,
	}
	if err := conn.Authenticate(authClient); err != nil {
		return fmt.Errorf("authentication attempt: %w", err)
	}
	return nil
}

func (c *clientXOAUTH2) Start() (mech string, ir []byte, err error) {
	str := fmt.Sprintf("user=%s\001auth=Bearer %s\001\001", c.email, c.token)
	ir = []byte(str)
	return "XOAUTH2", ir, nil
}

func (c *clientXOAUTH2) Next(challenge []byte) (response []byte, err error) {
	return nil, nil
}

func (c Client) FetchFolders(ctx context.Context) ([]mail.Folder, error) {
	conns, err := c.createConnections(ctx, c.config.MaxConnections)
	if err != nil {
		return nil, fmt.Errorf("create connections: %w", err)
	}

	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()
	
	cmd := conns[0].List("", "*", &imap.ListOptions{ReturnStatus: &imap.StatusOptions{UIDNext: true, UIDValidity: true}})
	defer cmd.Close()

	data, err := cmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("collect command: %w", err)
	}
	result := make([]mail.Folder, len(data))
	for i, item := range data {
		result[i] = mail.Folder{
			Name:        item.Mailbox,
			UIDNext:     uint32(item.Status.UIDNext),
			UIDValidity: item.Status.UIDValidity,
		}
	}
	return result, nil
}

func (c Client) FetchNewLetters(ctx context.Context, folder string, uid uint32) ([]mail.Letter, error) {
	conns, err := c.createConnections(ctx, int(c.config.MaxConnections))
	if err != nil {
		return nil, fmt.Errorf("create connections: %w", err)
	}

	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	type response struct {
		letters []mail.Letter
		length  int
	}

	var result []mail.Letter
	clices := uint32(len(conns))
	chRes := make(chan response, clices)
	chErr := make(chan error, clices)

	startSeq, err := c.getMessageSeqNum(conns[0], folder, uid)
	if err != nil {
		return nil, fmt.Errorf("get start message seq num: %w", err)
	}
	stopSeq, err := c.getMessageSeqNum(conns[0], folder, uint32(1<<32-1))
	if err != nil {
		return nil, fmt.Errorf("get stop message seq num: %w", err)
	}
	length := stopSeq - startSeq + 1

	if length < 10*clices {
		res, _, err := c.getLetters(conns[0], folder, startSeq, stopSeq)
		if err != nil {
			return nil, fmt.Errorf("get letters from %s: %w", folder, err)
		}
		return res, nil
	}

	for i := range clices {
		start := startSeq + uint32(i)*(length/clices)
		stop := startSeq + uint32(i+1)*(length/clices) - 1
		if i == clices-1 {
			stop = stopSeq
		}

		go func() {
			res, length, err := c.getLetters(conns[i], folder, start, stop)
			if err != nil {
				chErr <- fmt.Errorf("get letters from %s: %w", folder, err)
				return
			}
			chRes <- response{res, length}
		}()
	}

	totalChars := 0

	for range clices {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case err := <-chErr:
			if ctx.Err() != nil {
				return result, ctx.Err()
			}
			return result, err
		case resp := <-chRes:
			totalChars += resp.length
			if totalChars > c.config.FolderCharsLimit {
				return result, nil
			}
			result = append(result, resp.letters...)
		}
	}
	return result, nil
}

func (c Client) createConnections(ctx context.Context, num int) ([]*imapclient.Client, error) {
	conns := make([]*imapclient.Client, 0, num)

	for range num {
		conn, err := c.connect(ctx)
		if err != nil {
			continue
		}
		if err := c.authenticate(conn); err != nil {
			conn.Close()
			continue
		}
		conns = append(conns, conn)
	}

	if len(conns) == 0 {
		return nil, fmt.Errorf("failed to create at least 1 connection")
	}
	return conns, nil
}

func (c Client) getLetters(conn *imapclient.Client, folder string, start uint32, stop uint32) ([]mail.Letter, int, error) {
	var letters []mail.Letter
	lenght := 0
	buffer, err := c.getMessages(conn, folder, start, stop)
	if err != nil {
		return nil, 0, fmt.Errorf("get messages from %s: %w", folder, err)
	}

	for _, msg := range buffer {
		if len(msg.Envelope.From) != 0 && inBlackList(msg.Envelope.From[0].Mailbox) {
			continue
		}
		body, _ := getMessageBody(msg)
		if body == "" {
			continue
		}
		body = body[:min(len(body), c.config.LetterCharsLimit)]
		lenght += len(body)
		if lenght > c.config.FolderCharsLimit {
			break
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
	return letters, lenght, nil
}

func (c Client) getMessageSeqNum(conn *imapclient.Client, folder string, uid uint32) (uint32, error) {
	_, err := conn.Select(folder, nil).Wait()
	if err != nil {
		return 0, fmt.Errorf("select command: %w", err)
	}

	uidSet := imap.UIDSet{}
	uidSet.AddRange(imap.UID(uid), 0)
	data, err := conn.Search(&imap.SearchCriteria{UID: []imap.UIDSet{uidSet}}, nil).Wait()
	if err != nil {
		return 0, fmt.Errorf("search seq number by uid: %w", err)
	}

	seqNums := data.AllSeqNums()
	if len(seqNums) == 0 {
		return 0, nil
	}

	return seqNums[0], nil
}

func (c Client) getMessages(conn *imapclient.Client, folder string, start uint32, stop uint32) ([]*imapclient.FetchMessageBuffer, error) {
	mailbox, err := conn.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("select command: %w", err)
	}
	if mailbox.NumMessages == 0 {
		return nil, nil
	}

	seqSet := imap.SeqSet{}
	seqSet.AddRange(start, stop)

	messages, err := conn.Fetch(seqSet, &imap.FetchOptions{
		Envelope:    true,
		UID:         true,
		BodySection: []*imap.FetchItemBodySection{{}},
	}).Collect()

	if err != nil {
		return nil, fmt.Errorf("fetch command: %w", err)
	}
	return messages, nil
}

// getMessageBody extracts and returns text/plain data from an IMAP message
func getMessageBody(message *imapclient.FetchMessageBuffer) (string, error) {
	body := message.FindBodySection(&imap.FetchItemBodySection{})
	mr, err := imapmail.CreateReader(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create reader: %w", err)
	}

	var htmlText string

	for p, err := mr.NextPart(); err != io.EOF; {
		if err != nil {
			return "", fmt.Errorf("next part: %w", err)
		}
		dataType := p.Header.Get("Content-Type")

		switch {
		case strings.HasPrefix(dataType, "text/plain"):
			body, _ := io.ReadAll(p.Body)
			return cleanPlainText(string(body)), nil
		case strings.HasPrefix(dataType, "text/html"):
			body, _ := io.ReadAll(p.Body)
			htmlText = htmlToText(string(body))
		}
		p, err = mr.NextPart()
	}
	return htmlText, nil
}

// cleanPlainText clears raw string of unnecessary information
func cleanPlainText(raw string) string {
	text := reText.ReplaceAllString(raw, " ")
	text = removeNotPrintable(text)
	text = strings.ToValidUTF8(text, "")
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

func inBlackList(mailbox string) bool {
	return strings.Contains(mailbox, "noreply") ||
		strings.Contains(mailbox, "no-reply") ||
		strings.Contains(mailbox, "devnull") ||
		strings.Contains(mailbox, "robot")
}
