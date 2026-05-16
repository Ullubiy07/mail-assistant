package storage

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"backend/mail/internal/client/embed"
	"backend/mail/internal/client/imap"
)

type Point struct {
	Embedding embed.Embedding
	Payload   *imap.Letter
}

type ScoredPoint struct {
	Score   float32
	Payload *imap.Letter
}

type Mailbox struct {
	UserID  uuid.UUID
	Email   string
	Folders []imap.Folder
}

var (
	ErrNotFoundFolders = errors.New("folders not found")
)

type VectorStorage interface {
	Insert(ctx context.Context, userID uuid.UUID, points []Point) error
	Search(ctx context.Context, userID uuid.UUID, embedding embed.Embedding) ([]ScoredPoint, error)
}

type MailStorage interface {
	GetFolders(ctx context.Context, userID uuid.UUID, email string) ([]imap.Folder, error)
	UpdateMailbox(ctx context.Context, mailbox Mailbox) error
}
