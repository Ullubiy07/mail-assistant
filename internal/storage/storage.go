package storage

import (
	"context"
	"errors"
	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/model"
)

type Point struct {
	Embedding embed.Embedding
	Payload   *mail.Letter
}

type ScoredPoint struct {
	Score   float32
	Payload *mail.Letter
}

var (
	ErrDublicateUser = errors.New("email or username already exists")
	ErrNotFoundUser  = errors.New("user not found")
)

type VectorStorer interface {
	CreateCollection(ctx context.Context, collName string) error
	DeleteCollection(ctx context.Context, collName string) error
	Upsert(ctx context.Context, collName string, points []Point) error
	Search(ctx context.Context, name string, embedding embed.Embedding) ([]ScoredPoint, error)
}

type UserStorer interface {
	CreateUser(ctx context.Context, user model.UserRegister) error
	FindUserByUsername(ctx context.Context, username string) (model.User, error)
}
