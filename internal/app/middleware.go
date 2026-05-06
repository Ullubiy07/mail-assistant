package app

import (
	"context"
	"log/slog"
	"mail-assistant/internal/pkg/metrics"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type LogResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type Middleware struct {
	handler http.Handler
}

func (w *LogResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/metrics" {
		m.handler.ServeHTTP(w, r)
		return
	}

	requestId := uuid.NewString()
	r.Header.Add("X-Request-ID", requestId)
	w.Header().Set("X-Request-ID", requestId)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	lrw := &LogResponseWriter{w, http.StatusOK}

	m.handler.ServeHTTP(lrw, r.WithContext(ctx))

	slog.Info("HTTP request", "method", r.Method, "path", r.URL.Path, "status", lrw.statusCode, "request_id", requestId)

	metrics.HttpRequestByPath.WithLabelValues(r.URL.Path).Inc()
	metrics.HttpRequestDuration.WithLabelValues(r.Method + " " + r.URL.Path)
}
