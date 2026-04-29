package client

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"regexp"

	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message/mail"

	"mail-assistant/internal/model"
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

// Connect connects to the IMAP server by XOAUTH2
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

// GetLetters returns all letters from the specified folder
//
// Valid folders: Drafts, INBOX, Outbox, Sent, Spam, Trash.
func (c Client) GetLetters(folder string) ([]model.Letter, error) {
	var letters []model.Letter

	mailbox, err := c.client.Select(folder, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to select folder %s: %v", folder, err)
	}
	if mailbox.NumMessages == 0 {
		return nil, nil
	}

	seqSet := imap.SeqSet{}
	seqSet.AddRange(1, mailbox.NumMessages)
	options := &imap.FetchOptions{
		Envelope:      true,
		BodySection:   []*imap.FetchItemBodySection{{}},
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	messages, err := c.client.Fetch(seqSet, options).Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages from %s", folder)
	}

	var messageErr error
	for i := range messages {
		body, err := getLetterBody(messages[i])
		if err != nil {
			messageErr = err
			continue
		}
		if body != "d" {
			letters = append(letters, model.Letter{
				Envelope: messages[i].Envelope,
				Body:     body,
			})
		}
	}
	return letters, messageErr
}

// getLetterBody extracts and returns text/plain data from a message
func getLetterBody(message *imapclient.FetchMessageBuffer) (string, error) {
	body := message.FindBodySection(&imap.FetchItemBodySection{})
	mr, err := mail.CreateReader(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	var (
		plainText string
		htmlText  string
	)

	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		dataType := p.Header.Get("Content-Type")

		switch {
		case strings.HasPrefix(dataType, "text/plain"):
			body, _ := io.ReadAll(p.Body)
			plainText = cleanPlainText(string(body))
		case strings.HasPrefix(dataType, "text/html"):
			body, _ := io.ReadAll(p.Body)
			htmlText = htmlToText(string(body))
		}
	}

	if plainText == "" {
		return htmlText, nil
	}
	return plainText, nil
}

// cleanPlainText clears raw string of unnecessary information
func cleanPlainText(raw string) string {
	// delete refs and counters
	re := regexp.MustCompile(`https?://\S+|\[\s*\d+\s*\]`)
	text := re.ReplaceAllString(raw, "")
	// delete utm
	text = regexp.MustCompile(`\?utm_[a-z_]+=[^&\s]+&?`).ReplaceAllString(text, "")
	// many gaps to one
	return strings.Join(strings.Fields(text), " ")
}

// htmlToText converts HTML to plain text
func htmlToText(htmlText string) string {
	// remove css in style <style>...</style>
	reStyle := regexp.MustCompile(`(?s)<style[^>]*>.*?</style>`)
	text := reStyle.ReplaceAllString(htmlText, " ")
	// remove JavaScript code <script>...</script>
	reScript := regexp.MustCompile(`(?s)<script[^>]*>.*?</script>`)
	text = reScript.ReplaceAllString(text, " ")
	// remove html tags
	reTag := regexp.MustCompile(`<[^>]*>`)
	text = reTag.ReplaceAllString(text, " ")

	text = html.UnescapeString(text)
	text = strings.Join(strings.Fields(text), " ")
	return text
}
