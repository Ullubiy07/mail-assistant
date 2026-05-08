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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	connUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres", config.Database.User, config.Database.Password, config.Database.Host, config.Database.Port)
	db, err := pgxpool.New(context.TODO(), connUrl)
	if err != nil {
		return App{}, fmt.Errorf("new database pool: %w", err)
	}
	defer db.Close()
	
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

	userStorage := postgres.NewUserStorage(db)
	mailStorage := postgres.NewMailStorage(db)

	auth := handler.NewAuthHandler(userStorage, jwtGenerator)
	mail := handler.NewMailHandler(mailStorage, qdrantClient, imapFactory, gigachat, jwtVerifier)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("POST /api/register", auth.Register)
	mux.HandleFunc("POST /api/login", auth.Login)
	mux.HandleFunc("POST /api/mail/ask", mail.Question)

	app := App{
		server: &http.Server{
			Addr:         ":" + config.App.ServerPort,
			Handler:      Middleware{handler: mux},
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
	if err := a.server.Close(); err != nil {
		return fmt.Errorf("close server: %w", err)
	}
	return nil
}
