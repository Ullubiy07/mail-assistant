package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"

	"mail-assistant/internal/app"
	"mail-assistant/internal/config"
	"mail-assistant/internal/pkg/logger"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		slog.Error("config new", "err", err)
		os.Exit(1)
	}

	logger := logger.New(cfg.Log.Mode)
	slog.SetDefault(logger)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", uuid.NewString())
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
	defer cancel()

	// imap := imap.New(&cfg.IMAP, "PLAIN", "imap.mail.ru:993", "saygitov07@xmail.ru", "Z2oK78DS1AvqNAfcMqcy", "")
	// letters, state, err := imap.GetNewLetters(ctx, "INBOX", 3600)

	// for _, item := range letters {
	// 	fmt.Println(item.Envelope.Subject, item.Envelope.From)
	// }
	// fmt.Println(len(letters), state)

	app, err := app.New(&cfg.App)
	if err != nil {
		slog.Error("Creating a new application", "error", fmt.Errorf("app new: %w", err))
		os.Exit(1)
	}

	defer app.Stop()

	go func() {
		if err := app.Run(); err != nil {
			slog.Error("Starting the application", "error", fmt.Errorf("app run: %w", err))
			return
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
}
