package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"mail-assistant/internal/client/embed/gigachat"
	"mail-assistant/internal/client/mail/imap"
	"mail-assistant/internal/config"
	"mail-assistant/internal/handler"
	"mail-assistant/internal/storage/postgres"
	"mail-assistant/internal/storage/qdrant"
	"mail-assistant/internal/token/jwt"
)

type App struct {
	server *http.Server
	db     *pgxpool.Pool
}

func New(config *config.Config) (App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres", config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port)
	db, err := pgxpool.New(ctx, connUrl)
	if err != nil {
		return App{}, fmt.Errorf("new database pool: %w", err)
	}

	if err = db.Ping(ctx); err != nil {
		return App{}, fmt.Errorf("ping database: %w", err)
	}

	qdrantClient, err := qdrant.New(config.Qdrant)
	if err != nil {
		return App{}, fmt.Errorf("qdrant new client: %w", err)
	}
	gigachat := gigachat.New(config.Embedding)
	imapFactory := imap.New(config.IMAP)
	jwtGenerator := jwt.NewGenerator(config.Token)
	jwtVerifier := jwt.NewVerifier(config.Token)
	jwtExtractor := jwt.NewExtractor()

	userStorage := postgres.NewUserStorage(db)
	mailStorage := postgres.NewMailStorage(db)

	auth := handler.NewAuthHandler(userStorage, jwtGenerator)
	mail := handler.NewMailHandler(mailStorage, qdrantClient, imapFactory, gigachat)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("POST /auth/register", auth.Register)
	mux.HandleFunc("POST /auth/login", auth.Login)
	mux.HandleFunc("POST /mail/ask", mail.Question)

	app := App{
		server: &http.Server{
			Addr:         ":" + config.App.ServerPort,
			Handler:      handler.Middleware{Handler: mux, Verifier: jwtVerifier, Extractor: jwtExtractor},
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		db: db,
	}

	return app, nil
}

func (a *App) Run() error {
	slog.Info("started server process [" + strconv.Itoa(os.Getpid()) + "]")
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (a *App) Stop() error {
	slog.Info("stopping server...")
	a.db.Close()
	if err := a.server.Close(); err != nil {
		return fmt.Errorf("close server: %w", err)
	}
	return nil
}
