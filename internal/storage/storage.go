package storage

import (
	"context"
	"errors"
	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/model"

	"github.com/google/uuid"
)

type Point struct {
	Embedding embed.Embedding
	Payload   *mail.Letter
}

type ScoredPoint struct {
	Score   float32
	Payload *mail.Letter
}

type Mailbox struct {
	UserID  uuid.UUID
	Email   string
	Folders []mail.Folder
}

var (
	ErrDublicateUser   = errors.New("email or username already exists")
	ErrNotFoundUser    = errors.New("user not found")
	ErrNotFoundFolders = errors.New("folders not found")
)

type VectorStorage interface {
	CreateCollection(ctx context.Context, collName string) error
	Insert(ctx context.Context, collName string, points []Point) error
	Search(ctx context.Context, name string, embedding embed.Embedding) ([]ScoredPoint, error)
}

type UserStorage interface {
	CreateUser(ctx context.Context, user model.UserRegister) error
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
}

type MailStorage interface {
	GetFolders(ctx context.Context, userID uuid.UUID, email string) ([]mail.Folder, error)
	UpdateMailbox(ctx context.Context, mailbox Mailbox) error
}
