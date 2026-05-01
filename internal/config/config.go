package config

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	TokenAuthURL string `env:"TOKEN_AUTH_URL, required"`
	TokenAuthKey string `env:"TOKEN_AUTH_KEY, required"`
	EmbeddingURL string `env:"EMBEDDING_URL, required"`
}

func New() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}
	ctx := context.Background()

	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
