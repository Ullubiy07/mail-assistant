package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MailStorage struct {
	db *pgxpool.Pool
}

func NewMailStorage(db *pgxpool.Pool) MailStorage {
	return MailStorage{db: db}
}

func (s MailStorage) UpdateMailbox(ctx context.Context, mailbox storage.Mailbox) error {
	query := `
		INSERT INTO mailboxes (user_id, email, folders)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, email) DO UPDATE
		SET folders = $3
	`
	raw, err := json.Marshal(mailbox.Folders)
	if err != nil {
		return fmt.Errorf("marshal folders: %w", err)
	}
	if _, err := s.db.Exec(ctx, query, mailbox.UserID, mailbox.Email, raw); err != nil {
		return fmt.Errorf("db exec query: %w", err)
	}
	return nil
}

func (s MailStorage) GetFolders(ctx context.Context, userID uuid.UUID, email string) ([]mail.Folder, error) {
	query := `
		SELECT folders FROM mailboxes
		WHERE user_id = $1 AND email = $2
	`
	var (
		raw     []byte
		folders []mail.Folder
	)

	err := s.db.QueryRow(ctx, query, userID, email).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFoundFolders
		}
		return nil, fmt.Errorf("query row: %w", err)
	}

	if err := json.Unmarshal(raw, &folders); err != nil {
		return nil, fmt.Errorf("unmarshal folders: %w", err)
	}
	return folders, nil
}
