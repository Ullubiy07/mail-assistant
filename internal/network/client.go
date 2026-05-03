package network

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	client *http.Client
}

func New(timeout time.Duration, transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{
		client: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
	}
}

func (c *Client) PostRequest(ctx context.Context, data []byte, URL string, header map[string]string) ([]byte, error) {
	reader := bytes.NewReader(data)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status not 2xx: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
