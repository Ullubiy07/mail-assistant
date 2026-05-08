package llm

import (
	"context"
	"mail-assistant/internal/storage"
)

type Response struct {}

type Prompter interface {
	Prompt(ctx context.Context, score []storage.ScoredPoint) (Response, error)
}
