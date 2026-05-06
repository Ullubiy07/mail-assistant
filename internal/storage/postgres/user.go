package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgconn"

	"mail-assistant/internal/model"
	"mail-assistant/internal/storage"
)

type UserStorage struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) UserStorage {
	return UserStorage{db: db}
}

func (s UserStorage) CreateUser(ctx context.Context, user model.UserRegister) error {
	query := `
		INSERT INTO users (id, username, email, password, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := s.db.Exec(ctx, query, uuid.New(), user.Username, user.Email, user.Password, time.Now())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return storage.ErrDublicateUser
		}
		return fmt.Errorf("db exec query: %w", err)
	}
	return nil
}

func (s UserStorage) FindUserByUsername(ctx context.Context, username string) (model.User, error) {
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
