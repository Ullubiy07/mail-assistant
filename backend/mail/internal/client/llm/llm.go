package llm

import (
	"context"

	"backend/mail/internal/client/imap"
	"backend/mail/internal/storage"
)

type Generator interface {
	Generate(ctx context.Context, question string, scored []storage.ScoredPoint) (string, error)
}

type Filterer interface {
	Filter(ctx context.Context, letters []imap.Letter) ([]imap.Letter, error)
}
