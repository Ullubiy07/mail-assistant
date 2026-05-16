package gigachatllm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"backend/mail/internal/config"
	"backend/mail/internal/storage"
	"backend/pkg/network"
)

type Client struct {
	client network.Client
	config config.LLM

	mu             sync.Mutex
	accessToken    string
	tokenExpiresAt time.Time
}

const systemPrompt = "Ты — личный ИИ-ассистент. Сначала развернуто и по-человечески " +
	"ответь на вопрос пользователя по тексту писем. Затем, если в письмах есть подтверждение, " +
	"строго ниже приложи карточку этого письма по структуре (полностью пропускай строки, если для них нет данных, никаких пустых полей):\n" +
	"**Тема: [Тема]**\n" +
	"Отправитель: [Имя/Email]\n" +
	"Дата: [Дата]\n" +
	"Содержание: [Краткая выжимка жирным шрифтом]\n" +
	"Ссылка: [URL-ссылка]"

func New(config config.LLM) *Client {
	httpClient := network.New(network.LoggingWrapper{
		Tripper: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Provider: "GigaChat",
	})
	return &Client{
		client: httpClient,
		config: config,
	}
}

func (c *Client) Generate(ctx context.Context, question string, scored []storage.ScoredPoint) (string, error) {
	var userPrompt strings.Builder
	userPrompt.WriteString("Вопрос пользователя: ")
	userPrompt.WriteString(question)
	userPrompt.WriteString("\n\n")

	for _, item := range scored {
		userPrompt.WriteString(fmt.Sprintf(
			"### Письмо (UID: %d)\n"+
				"- [SCORE]: %.2f\n"+
				"- [ОТ]: %s\n"+
				"- [ДАТА]: %s\n"+
				"- [ТЕМА]: %s\n"+
				"- [ТЕКСТ]: %s\n\n",
			item.Payload.Envelope.UID,
			item.Score,
			item.Payload.Envelope.From,
			item.Payload.Envelope.Date.Format("2006-01-02 15:04:05"),
			item.Payload.Envelope.Subject,
			item.Payload.Body,
		))
	}

	req := llmRequest{
		Model: "GigaChat",
		Messages: []message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt.String(),
			},
		},
	}
	resp := llmResponse{}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}

	if err := c.updateAccessToken(ctx); err != nil {
		return "", fmt.Errorf("update access token: %w", err)
	}

	raw, err := c.client.PostRequest(ctx, body, c.config.Endpoint, map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + c.accessToken,
	})
	if err != nil {
		return "", fmt.Errorf("post request to llm: %w", err)
	}

	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty llm content response")
	}

	return resp.Choices[0].Messages.Content, nil
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
	res, err := c.client.PostRequest(ctx, data, c.config.AuthURL, map[string]string{
		"Content-Type":  "application/x-www-form-urlencoded",
		"Accept":        "application/json",
		"RqUID":         uuid.NewString(),
		"Authorization": "Basic " + c.config.AuthKey,
	})
	if err != nil {
		return fmt.Errorf("http POST request: %w", err)
	}

	resp := tokenResponse{}
	if err = json.Unmarshal(res, &resp); err != nil {
		return fmt.Errorf("unmarshal http response: %w", err)
	}
	c.accessToken = resp.AccessToken
	c.tokenExpiresAt = time.Unix(int64(resp.ExpiresAt), 0)

	return nil
}
