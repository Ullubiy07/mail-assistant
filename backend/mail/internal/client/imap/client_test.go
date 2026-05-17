package imap

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"backend/mail/internal/config"
)

func TestFetchNewLetters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	factory := New(config.IMAP{LetterCharsLimit: 1500, MaxConnections: 10, FolderCharsLimit: 4000000})
	auth := Auth{
		Address:  "imap.mail.ru:993",
		Method:   "PLAIN",
		Email:    "saygitov07@xmail.ru",
		Password: "",
		Token:    "",
	}
	fetcher, err := factory.NewFetcher(ctx, auth, nil)
	require.NoError(t, err, "new fetcher")
	defer fetcher.Close()

	res, err := fetcher.FetchNewLetters(ctx, "INBOX", 5000)
	require.NoError(t, err, "fetch new letters")

	count := 0
	for _, item := range res {
		t.Log(item.Envelope.Subject)
		count += len(item.Body)
	}

	avg := 0
	if len(res) != 0 {
		avg = count / len(res)
	}
	t.Logf("Total: %d, Chars: %d, Average: %d", len(res), count, avg)


	// res, err := fetcher.FetchFolders(ctx)
	// require.NoError(t, err, "fetch folders")
	// t.Log(res)
}
