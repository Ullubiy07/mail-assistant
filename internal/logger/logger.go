package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/Marlliton/slogpretty"
)

type TraceHandler struct {
	slog.Handler
}

type LoggingMode = string

const (
	Development LoggingMode = "Dev"
	Production  LoggingMode = "Prod"
)

func (t *TraceHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value("trace_id").(string); ok {
		r.AddAttrs(slog.String("trace_id", id))
	}
	return t.Handler.Handle(ctx, r)
}

func New(mode LoggingMode) *slog.Logger {
	var logger *slog.Logger

	switch mode {
	case Development:
		logger = slog.New(&TraceHandler{
			Handler: slogpretty.New(os.Stdout, &slogpretty.Options{
				Level:      slog.LevelInfo,
				AddSource:  false,
				Colorful:   true,
				Multiline:  true,
				TimeFormat: "[02.01.06 15:04:05]",
			}),
		})
	case Production:
		logger = slog.New(&TraceHandler{
			Handler: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		})
	}
	return logger
}
