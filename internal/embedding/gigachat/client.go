package gigachat

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	client   *http.Client
	embURL   string
	tokenURL string
	authKey  string

	mu             sync.Mutex
	tokenExpiresAt time.Time
}

func New(tokenURL, embURL, authKey string) *Client {
	httpClient := &http.Client{
		Transport: &loggingWrapper{
			tripper: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
	return &Client{
		tokenURL: tokenURL,
		embURL:   embURL,
		authKey:  authKey,
		client:   httpClient,
	}
}

func (c *Client) Get(chunks []Chunk) ([]Embedding, error) {
	data := embeddingRequest{"Embeddings", chunks}
	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
	}

	accessToken, err := c.GetAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	res, err := c.postRequest(dataJson, c.embURL, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return nil, fmt.Errorf("request was not completed successfully: %w", err)
	}

	resp := embeddingResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return nil, fmt.Errorf("failed to unserialize the response: %w", err)
	}
	slog.Info("[GigaChat] Embeddings generated: " + strconv.Itoa(len(resp.Data)))

	result := make([]Embedding, len(resp.Data))
	for i, item := range resp.Data {
		result[i] = item.Embedding
	}

	return result, nil
}

func (c *Client) postRequest(data []byte, URL string, header map[string]string) ([]byte, error) {
	reader := bytes.NewReader(data)
	req, err := http.NewRequest(http.MethodPost, URL, reader)
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

func (c *Client) GetAccessToken() (string, error) {
	if time.Now().Add(time.Minute).Before(c.tokenExpiresAt) {
		return c.tokenExpiresAt.String(), nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	data := []byte("scope=GIGACHAT_API_PERS")
	res, err := c.postRequest(data, c.tokenURL, map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
		"RqUID":         uuid.NewString(),
		"Authorization": "Basic " + c.authKey,
	})
	if err != nil {
		return "", err
	}
	slog.Info("[GigaChat] Access token recieved")

	resp := tokenResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return "", err
	}
	return resp.AccessToken, nil
}
