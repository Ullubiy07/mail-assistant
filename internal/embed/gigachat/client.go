package gigachat

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	"mail-assistant/internal/config"
	"mail-assistant/internal/embed"
	"mail-assistant/internal/network"
)

type Client struct {
	client *network.Client
	cfg    *config.Embedding

	mu             sync.Mutex
	AccessToken    string
	tokenExpiresAt time.Time
}

func New(cfg *config.Embedding) *Client {
	httpClient := network.New(time.Duration(cfg.HttpTimeout)*time.Second, &loggingWrapper{
		tripper: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	})
	return &Client{
		cfg:    cfg,
		client: httpClient,
	}
}

func (c *Client) Embed(ctx context.Context, chunks []embed.Chunk) ([]embed.Embedding, error) {
	body, err := json.Marshal(embeddingRequest{"Embeddings", chunks})
	if err != nil {
		return nil, fmt.Errorf("serialize data: %w", err)
	}

	if err = c.updateAccessToken(ctx); err != nil {
		return nil, fmt.Errorf("update access token: %w", err)
	}

	res, err := c.client.PostRequest(ctx, body, c.cfg.HandleURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.AccessToken,
	})
	if err != nil {
		return nil, fmt.Errorf("http POST request: %w", err)
	}

	resp := embeddingResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return nil, fmt.Errorf("unserialize the response: %w", err)
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
	res, err := c.client.PostRequest(ctx, data, c.cfg.TokenAuthURL, map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
		"RqUID":         uuid.NewString(),
		"Authorization": "Basic " + c.cfg.TokenAuthKey,
	})
	if err != nil {
		return fmt.Errorf("http POST request: %w", err)
	}

	resp := tokenResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return fmt.Errorf("unmarshal http response: %w", err)
	}
	c.AccessToken = resp.AccessToken
	c.tokenExpiresAt = time.Unix(int64(resp.ExpiresAt), 0)

	return nil
}
