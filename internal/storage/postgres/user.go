package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"mail-assistant/internal/model"
	"mail-assistant/internal/storage"
)

type UserStorage struct {
	db *pgxpool.Pool
}

func NewUserStorage(db *pgxpool.Pool) UserStorage {
	return UserStorage{db: db}
}

func (s UserStorage) CreateUser(ctx context.Context, user model.UserRegister) error {
	query := `
		INSERT INTO users (username, email, password)
		VALUES ($1, $2, $3)
	`
	_, err := s.db.Exec(ctx, query, user.Username, user.Email, user.Password)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return storage.ErrDublicateUser
		}
		return fmt.Errorf("db exec query: %w", err)
	}
	return nil
}

func (s UserStorage) GetUserByUsername(ctx context.Context, username string) (model.User, error) {
	query := `
		SELECT id, username, email, password FROM users
		WHERE username = $1
	`
	user := model.User{}

	err := s.db.QueryRow(ctx, query, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password)
	if err != nil {
		// var pgErr *pgconn.PgError
		// if errors.As(err, &pgErr) &&
		return model.User{}, fmt.Errorf("db query row: %w", err)
	}
	return user, nil
}
