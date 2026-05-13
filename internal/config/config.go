package config

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Embedding struct {
	TokenAuthURL string `env:"TOKEN_AUTH_URL"`
	TokenAuthKey string `env:"TOKEN_AUTH_KEY"`
	Endpoint     string `env:"ENDPOINT"`
}

type IMAP struct {
	LetterCharsLimit int    `env:"LETTER_CHARS_LIMIT"`
	FolderCharsLimit int    `env:"FOLDER_CHARS_LIMIT"`
	DialTimeout      int    `env:"DIAL_TIMEOUT"`
	MaxConnections   int    `env:"MAX_CONNECTIONS"`
}

type Qdrant struct {
	Host          string `env:"QDRANT_HOST"`
	Port          int    `env:"QDRANT_PORT"`
	ApiKey        string `env:"QDRANT_API_KEY"`
	EmbeddingSize int    `env:"EMBEDDING_SIZE"`
}

type Log struct {
	Mode string `env:"MODE"`
}

type App struct {
	ServerPort string `env:"SERVER_PORT"`
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
	Database  Database
	Token     Token
}

func New(envPath string) (*Config, error) {
	if err := godotenv.Load(envPath); err != nil {
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
