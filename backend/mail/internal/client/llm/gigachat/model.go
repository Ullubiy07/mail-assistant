package gigachatllm

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int    `json:"expires_at"`
}

type llmRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type llmResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Messages message `json:"message"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
