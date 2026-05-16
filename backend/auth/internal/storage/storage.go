package storage

import (
	"context"
	"errors"

	"backend/auth/internal/model"
)

var (
	ErrDublicateUser = errors.New("email or username already exists")
	ErrNotFoundUser  = errors.New("user not found")
)

type UserStorage interface {
	CreateUser(ctx context.Context, user model.UserRegister) error
	GetUserByUsername(ctx context.Context, username string) (model.User, error)
}
