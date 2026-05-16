package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"backend/mail/internal/token"
	"backend/pkg/metrics"
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

	if !m.authenticate(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	lrw := &LogResponseWriter{w, http.StatusOK}

	requestId := uuid.NewString()
	r.Header.Add("X-Request-ID", requestId)
	w.Header().Set("X-Request-ID", requestId)

	jwt := strings.Split(r.Header.Get("Authorization"), "Bearer ")[1]
	user, err := m.Extractor.Extract(jwt)
	if err != nil {
		slog.Error("extract user from jwt", "error", err)
		sendResponse(w, http.StatusInternalServerError, "Internal error")
		return
	}

	ctx = context.WithValue(ctx, "user_id", user.Sub)

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
