package app

import (
	"mail-assistant/internal/config"
)

type App struct {
	cfg *config.Config
}

func New(cfg *config.Config) (*App, error) {

	return &App{
		cfg: cfg,
	}, nil
}
