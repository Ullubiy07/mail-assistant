package config

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type App struct {
	ServerPort string `env:"MAIL_SERVER_PORT"`
}

type Database struct {
	URL string `env:"DB_URL"`
}

type Qdrant struct {
	Host              string `env:"QDRANT_HOST"`
	Port              int    `env:"QDRANT_PORT"`
	ApiKey            string `env:"QDRANT_API_KEY"`
	EmbeddingSize     int    `env:"EMBEDDING_SIZE"`
	SearchPointsLimit uint64 `env:"SEARCH_POINTS_LIMIT"`
	CollectionName    string `env:"COLLECTION_NAME"`
}

type Embedding struct {
	AuthURL  string `env:"GIGACHAT_AUTH_URL"`
	AuthKey  string `env:"GIGACHAT_AUTH_KEY"`
	Endpoint string `env:"GIGACHAT_EMBED_ENDPOINT"`
	Model    string `env:"GIGACHAT_EMBED_MODEL"`
}

type LLM struct {
	AuthURL  string `env:"GIGACHAT_AUTH_URL"`
	AuthKey  string `env:"GIGACHAT_AUTH_KEY"`
	Endpoint string `env:"GIGACHAT_LLM_ENDPOINT"`
}

type IMAP struct {
	LetterCharsLimit int  `env:"LETTER_CHARS_LIMIT"`
	FolderCharsLimit int  `env:"FOLDER_CHARS_LIMIT"`
	MaxConnections   uint `env:"MAX_CONNECTIONS"`
}

type Token struct {
	SecretKey string `env:"JWT_SECRET_KEY"`
}

type Log struct {
	Mode string `env:"MODE"`
}

type Config struct {
	App       App
	Database  Database
	Qdrant    Qdrant
	Embedding Embedding
	LLM       LLM
	IMAP      IMAP
	Token     Token
	Log       Log
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
