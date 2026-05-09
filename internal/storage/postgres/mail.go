package postgres

import (
	"context"
	"mail-assistant/internal/client/mail"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MailStorage struct {
	db *pgxpool.Pool
}

func NewMailStorage(db *pgxpool.Pool) MailStorage {
	return MailStorage{db: db}
}

func (s MailStorage) CreateFolderRecord(ctx context.Context, state mail.Folder) error {
	return nil
}
