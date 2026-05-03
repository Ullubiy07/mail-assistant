package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	"mail-assistant/internal/config"
	"mail-assistant/internal/logger"
)

type TraceHandler struct {
	slog.Handler
}

func (t *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value("trace_id").(string); ok {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return t.Handler.Handle(ctx, r)
}

func main() {
	cfg, err := config.New()
	if err != nil {
		slog.Error("[Config] failed to load config", "err", err)
		os.Exit(1)
	}

	logger := logger.New(cfg.Log.Mode)
	slog.SetDefault(logger)

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", uuid.NewString())
	ctx, cancel := context.WithTimeout(ctx, 500*time.Second)
	defer cancel()

}
