package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"backend/auth/internal/config"
	"backend/auth/internal/handler"
	"backend/auth/internal/storage/postgres"
	"backend/auth/internal/token/jwt"
)

type App struct {
	server *http.Server
	db     *pgxpool.Pool
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

	jwtGenerator := jwt.NewGenerator(config.Token)
	userStorage := postgres.NewUserStorage(db)

	auth := handler.NewAuthHandler(userStorage, jwtGenerator)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("POST /auth/register", auth.Register)
	mux.HandleFunc("POST /auth/login", auth.Login)

	app := App{
		server: &http.Server{
			Addr:         ":" + config.App.ServerPort,
			Handler:      handler.Middleware{Handler: mux},
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		db: db,
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
	if err := a.server.Close(); err != nil {
		return fmt.Errorf("close server: %w", err)
	}
	return nil
}
