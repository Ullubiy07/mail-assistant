package network

import (
	"log/slog"
	"net/http"
	"time"
)

type LoggingWrapper struct {
	Tripper  http.RoundTripper
	Provider string
}

func (w LoggingWrapper) RoundTrip(req *http.Request) (*http.Response, error) {

	start := time.Now()
	ctx := req.Context()

	resp, err := w.Tripper.RoundTrip(req)

	slog.InfoContext(ctx, "HTTP Request",
		"provider", w.Provider,
		"url", req.URL.String(),
		"method", req.Method,
		"duration", time.Since(start).Round(time.Millisecond),
		"error", err,
	)

	return resp, err
}
