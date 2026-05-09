package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"mail-assistant/internal/pkg/metrics"
	"mail-assistant/internal/token"
)

type LogResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

type Middleware struct {
	Handler   http.Handler
	Verifier  token.Verifier
	Extractor token.Extractor
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

	if !strings.HasPrefix(r.URL.Path, "/auth") && !m.authenticate(w, r) {
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/auth") {
		jwt := strings.Split(r.Header.Get("Authorization"), "Bearer ")[1]
		user, err := m.Extractor.Extract(jwt)
		if err != nil {
			sendResponse(w, http.StatusInternalServerError, "Internal error")
			return
		}
		slog.Info("debug", "user", user)
	}

	requestId := uuid.NewString()
	r.Header.Add("X-Request-ID", requestId)
	w.Header().Set("X-Request-ID", requestId)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	lrw := &LogResponseWriter{w, http.StatusOK}

	m.Handler.ServeHTTP(lrw, r.WithContext(ctx))

	slog.Info("HTTP request", "method", r.Method, "path", r.URL.Path, "status", lrw.statusCode, "request_id", requestId)

	metrics.HttpRequestByPath.WithLabelValues(r.URL.Path).Inc()
	metrics.HttpRequestDuration.WithLabelValues(r.Method + " " + r.URL.Path)
}

func (m Middleware) authenticate(w http.ResponseWriter, r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		sendResponse(w, http.StatusUnauthorized, "Missing authorization header")
		return false
	}
	parts := strings.Split(authHeader, "Bearer ")
	if len(parts) != 2 {
		sendResponse(w, http.StatusUnauthorized, "Malformed token")
		return false
	}

	jwt := parts[1]
	err := m.Verifier.Verify(jwt)
	if errors.Is(err, token.ErrTokenExpired) {
		sendResponse(w, http.StatusUnauthorized, "Token has expired")
		return false
	} else if errors.Is(err, token.ErrInvalidSignature) {
		sendResponse(w, http.StatusUnauthorized, "Unauthorized")
		return false
	} else if err != nil {
		sendResponse(w, http.StatusUnauthorized, "Internal error")
		return false
	}
	return true
}
