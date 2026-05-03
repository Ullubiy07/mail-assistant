package embed

import "context"

type Chunk = string
type Embedding = []float32

type Embedder interface {
	Embed(ctx context.Context, chunks []Chunk) ([]Embedding, error)
}
