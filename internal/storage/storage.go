package storage

import (
	"context"
	"mail-assistant/internal/embed"
)

type Point struct {
	Embedding embed.Embedding
	Payload   map[string]any
}

type VectorStore interface {
	CreateCollection(ctx context.Context, collName string) error
	Upsert(ctx context.Context, collName string, points []Point) error
	Query(ctx context.Context, collName string, embedding embed.Embedding)
}