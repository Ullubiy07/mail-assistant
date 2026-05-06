package config

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Embedding struct {
	TokenAuthURL string `env:"TOKEN_AUTH_URL"`
	TokenAuthKey string `env:"TOKEN_AUTH_KEY"`
	HandleURL    string `env:"EMBEDDING_HANDLE_URL"`
	HttpTimeout  int    `env:"EMBEDDING_HTTP_TIMEOUT"`
}

type IMAP struct {
	LetterCharsLimit int `env:"LETTER_CHARS_LIMIT"`
	DialTimeout      int `env:"DIAL_TIMEOUT"`
}

type Qdrant struct {
	Host          string `env:"QDRANT_HOST"`
	Port          int    `env:"QDRANT_PORT"`
	API_KEY       string `env:"QDRANT_API_KEY"`
	EmbeddingSize int    `env:"EMBEDDING_SIZE"`
}

type Log struct {
	Mode string `env:"MODE"`
}

type App struct {
	ServerPort string `env:"SERVER_PORT"`
	Database   Database
	Token      Token
}

type Token struct {
	SecretKey string `env:"JWT_SECRET_KEY"`
}

type Database struct {
	Host     string `env:"POSTGRES_HOST"`
	Port     int    `env:"POSTGRES_PORT"`
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
}

type Config struct {
	Embedding Embedding
	IMAP      IMAP
	Qdrant    Qdrant
	Log       Log
	App       App
}

func New() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

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
