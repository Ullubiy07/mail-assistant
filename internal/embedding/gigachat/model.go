package gigachat

import "mail-assistant/internal/embedding"

type Embedding = embedding.Embedding
type Chunk = embedding.Chunk

type embeddingRequest struct {
	Model string  `json:"model"`
	Input []Chunk `json:"input"`
}

type embeddingResponse struct {
	Object string          `json:"object"`
	Model  string          `json:"model"`
	Data   []embeddingItem `json:"data"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int    `json:"expires_at"`
}

type embeddingItem struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding Embedding `json:"embedding"`
	Usage     usage     `json:"usage"`
}

type usage struct {
	PromptTokens int `json:"prompt_tokens"`
}
