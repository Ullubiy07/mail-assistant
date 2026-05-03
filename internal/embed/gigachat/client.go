package gigachat

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"mail-assistant/internal/config"
	"mail-assistant/internal/embed"
)

type Client struct {
	client *http.Client
	cfg    *config.Embedding

	mu             sync.Mutex
	AccessToken    string
	tokenExpiresAt time.Time
}

func New(cfg *config.Embedding) *Client {
	httpClient := &http.Client{
		Transport: &loggingWrapper{
			tripper: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
	return &Client{
		cfg:    cfg,
		client: httpClient,
	}
}

func (c *Client) Embed(ctx context.Context, chunks []embed.Chunk) ([]embed.Embedding, error) {
	body, err := json.Marshal(embeddingRequest{"Embeddings", chunks})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
	}

	if err = c.updateAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to update access token: %w", err)
	}

	res, err := c.postRequest(ctx, body, c.cfg.EmbeddingURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.AccessToken,
	})
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}

	resp := embeddingResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return nil, fmt.Errorf("failed to unserialize the response: %w", err)
	}

	result := make([]embed.Embedding, 0, len(resp.Data))
	for _, item := range resp.Data {
		result = append(result, item.Embedding)
	}

	return result, nil
}

func (c *Client) updateAccessToken(ctx context.Context) error {
	if time.Now().Add(time.Minute).Before(c.tokenExpiresAt) {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if time.Now().Add(time.Minute).Before(c.tokenExpiresAt) {
		return nil
	}

	data := []byte("scope=GIGACHAT_API_PERS")
	res, err := c.postRequest(ctx, data, c.cfg.TokenAuthURL, map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
		"RqUID":         uuid.NewString(),
		"Authorization": "Basic " + c.cfg.TokenAuthKey,
	})
	if err != nil {
		return err
	}

	resp := tokenResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return err
	}
	c.AccessToken = resp.AccessToken
	c.tokenExpiresAt = time.Unix(int64(resp.ExpiresAt), 0)

	return nil
}

func (c *Client) postRequest(ctx context.Context, data []byte, URL string, header map[string]string) ([]byte, error) {
	reader := bytes.NewReader(data)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, URL, reader)
	if err != nil {
		return nil, err
	}

	for key, value := range header {
		req.Header.Set(key, value)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get access token, status: %d, %s", resp.StatusCode, string(body))
	}

	return body, nil
}
