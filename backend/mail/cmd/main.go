package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"backend/mail/internal/app"
	"backend/mail/internal/config"
	"backend/pkg/logger"
)

func main() {
	config, err := config.New("mail/.env")
	if err != nil {
		slog.Error("config new", "err", err)
		os.Exit(1)
	}

	logger := logger.New(config.Log.Mode)
	slog.SetDefault(logger)

	app, err := app.New(config)
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
