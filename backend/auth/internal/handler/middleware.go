package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"backend/pkg/metrics"
)

type LogResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type Middleware struct {
	Handler http.Handler
}

func (w *LogResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/metrics" {
		m.Handler.ServeHTTP(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	lrw := &LogResponseWriter{w, http.StatusOK}

	requestId := uuid.NewString()
	r.Header.Add("X-Request-ID", requestId)
	w.Header().Set("X-Request-ID", requestId)

	m.Handler.ServeHTTP(lrw, r.WithContext(ctx))

	slog.Info("HTTP request", "method", r.Method, "path", r.URL.Path, "status", lrw.statusCode, "request_id", requestId)

	metrics.HttpRequestByPath.WithLabelValues(r.URL.Path).Inc()
	metrics.HttpRequestDuration.WithLabelValues(r.Method + " " + r.URL.Path)
}

func sendResponse(w http.ResponseWriter, statusCode int, msg string) {
	if w.Header().Get("Content-Type") == "" {
		w.Header().Add("Content-Type", "application/json")
	}
	w.WriteHeader(statusCode)

	raw, _ := json.Marshal(struct {
		Msg string `json:"message"`
	}{Msg: msg})

	w.Write(raw)
}
