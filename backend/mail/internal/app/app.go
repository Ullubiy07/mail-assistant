package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	gigachatembed "backend/mail/internal/client/embed/gigachat"
	"backend/mail/internal/client/imap"
	gigachatllm "backend/mail/internal/client/llm/gigachat"
	"backend/mail/internal/config"
	"backend/mail/internal/handler"
	"backend/mail/internal/storage/postgres"
	"backend/mail/internal/storage/qdrant"
	"backend/mail/internal/token/jwt"
)

type App struct {
	server *http.Server
	db     *pgxpool.Pool
	vector qdrant.Store
}

func New(config *config.Config) (App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, config.Database.URL)
	if err != nil {
		return App{}, fmt.Errorf("new database pool: %w", err)
	}

	if err = db.Ping(ctx); err != nil {
		return App{}, fmt.Errorf("ping database: %w", err)
	}

	qdrant, err := qdrant.New(ctx, config.Qdrant)
	if err != nil {
		return App{}, fmt.Errorf("qdrant new client: %w", err)
	}
	gigachatEmbedder := gigachatembed.New(config.Embedding)
	gigachatLLM := gigachatllm.New(config.LLM)
	imapFactory := imap.New(config.IMAP)
	jwtVerifier := jwt.NewVerifier(config.Token)
	jwtExtractor := jwt.NewExtractor()

	mailStorage := postgres.NewMailStorage(db)

	mail := handler.NewMailHandler(mailStorage, qdrant, imapFactory, gigachatLLM, gigachatEmbedder, config.IMAP)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("POST /mail/ask", mail.AnswerQuestion)
	mux.HandleFunc("POST /mail/folders", mail.GetFolders)

	app := App{
		server: &http.Server{
			Addr:         ":" + config.App.ServerPort,
			Handler:      handler.Middleware{Handler: mux, Verifier: jwtVerifier, Extractor: jwtExtractor},
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		db:     db,
		vector: qdrant,
	}

	return app, nil
}

func (a *App) Run() error {
	slog.Info(fmt.Sprintf("started server process on port [%s]", a.server.Addr))
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (a *App) Stop() error {
	slog.Info("stopping server...")
	a.db.Close()
	if err := a.vector.Close(); err != nil {
		return fmt.Errorf("close connection to vector storage: %w", err)
	}
	if err := a.server.Close(); err != nil {
		return fmt.Errorf("close server: %w", err)
	}
	return nil
}
