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

	"mail-assistant/internal/config"
	"mail-assistant/internal/handler"
	"mail-assistant/internal/storage/postgres"
)

type App struct {
	server *http.Server
	db     *pgxpool.Pool
}

func New(cfg *config.App) (App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
	defer cancel()

	connUrl := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres", cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port)
	db, err := pgxpool.New(context.TODO(), connUrl)
	if err != nil {
		return App{}, fmt.Errorf("new database pool: %w", err)
	}
	if err = db.Ping(ctx); err != nil {
		return App{}, fmt.Errorf("ping database: %w", err)
	}

	userStorage := postgres.New(db)

	auth := handler.NewAuthHandler(userStorage, &cfg.Token)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("POST /register", auth.Register)
	mux.HandleFunc("POST /login", auth.Login)
	mux.HandleFunc("POST /question", handler.Question)

	app := App{
		server: &http.Server{
			Addr:         ":" + cfg.ServerPort,
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
