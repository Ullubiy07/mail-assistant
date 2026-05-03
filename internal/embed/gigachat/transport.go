package gigachat

import (
	"log/slog"
	"net/http"
	"time"
)

type loggingWrapper struct {
	tripper http.RoundTripper
}

func (w *loggingWrapper) RoundTrip(req *http.Request) (*http.Response, error) {

	start := time.Now()
	ctx := req.Context()

	resp, err := w.tripper.RoundTrip(req)

	slog.InfoContext(ctx, "HTTP Request",
		"provider", "GigaChat",
		"url", req.URL.String(),
		"method", req.Method,
		"duration", time.Since(start).Round(time.Millisecond),
		"error", err,
	)

	return resp, err
}
