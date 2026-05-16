package config

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type App struct {
	ServerPort string `env:"AUTH_SERVER_PORT"`
}

type Database struct {
	URL string `env:"DB_URL"`
}

type Token struct {
	SecretKey string `env:"JWT_SECRET_KEY"`
}

type Log struct {
	Mode string `env:"MODE"`
}

type Config struct {
	App      App
	Database Database
	Token    Token
	Log      Log
}

func New(envPath string) (*Config, error) {
	_ = godotenv.Load(envPath)

	var cfg Config
	custom := envconfig.Config{
		Target:          &cfg,
		DefaultRequired: true,
	}

	if err := envconfig.Process(context.Background(), &custom); err != nil {
		return nil, err
	}
	return custom.Target.(*Config), nil
}
