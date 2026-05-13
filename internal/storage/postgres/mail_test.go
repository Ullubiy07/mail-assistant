package postgres

import (
	"context"
	"fmt"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/config"
	"mail-assistant/internal/storage"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMailbox(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	config, err := config.New("../../../.env")
	require.NoError(t, err, "config new")

	connUrl := fmt.Sprintf("postgres://%s:%s@localhost:5433/postgres", config.Database.User, config.Database.Password)
	db, err := pgxpool.New(ctx, connUrl)
	require.NoError(t, err, "new database pool")
	require.NoError(t, db.Ping(ctx), "ping database")

	s := NewMailStorage(db)

	userID := uuid.New()
	email1 := "john@gmail.com"
	email2 := "loh@yandex.ru"

	mailbox := []storage.Mailbox{
		{
			UserID: userID,
			Email:  email1,
			Folders: []mail.Folder{
				{Name: "INBOX", UIDNext: 3, UIDValidity: 23234},
				{Name: "Sent", UIDNext: 3, UIDValidity: 23435},
				{Name: "Outbox", UIDNext: 234, UIDValidity: 34}},
		},
		{
			UserID: userID,
			Email:  email1,
			Folders: []mail.Folder{
				{Name: "Trash", UIDNext: 33445, UIDValidity: 23}},
		},
		{
			UserID: userID,
			Email:  email2,
			Folders: []mail.Folder{
				{Name: "Folder", UIDNext: 9671, UIDValidity: 9345}},
		},
	}

	require.NoError(t, s.UpdateMailbox(ctx, mailbox[0]), "update mailbox")
	require.NoError(t, s.UpdateMailbox(ctx, mailbox[1]), "update mailbox")
	require.NoError(t, s.UpdateMailbox(ctx, mailbox[2]), "update mailbox")

	res1, err := s.GetFolders(ctx, userID, email1)
	require.NoError(t, err, "get folders")

	res2, err := s.GetFolders(ctx, userID, email2)
	require.NoError(t, err, "get folders")

	assert.Equal(t, res1, mailbox[1].Folders)
	assert.Equal(t, res2, mailbox[2].Folders)
}
