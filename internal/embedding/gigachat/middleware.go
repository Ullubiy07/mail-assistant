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

	resp, err := w.tripper.RoundTrip(req)

	slog.Info("[GigaChat]",
		"duration", time.Since(start).Round(time.Millisecond),
		"url", req.URL.String(),
		"method", req.Method,
		"error", nil,
	)

	return resp, err
}
