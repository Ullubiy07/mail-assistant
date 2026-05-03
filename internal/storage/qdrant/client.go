package qdrant

import (
	"context"
	"fmt"
	"mail-assistant/internal/config"
	"mail-assistant/internal/embed"
	"mail-assistant/internal/storage"

	"github.com/qdrant/go-client/qdrant"
)

type Client struct {
	client *qdrant.Client
	cfg    *config.Qdrant
}

func New(cfg *config.Qdrant) Client {
	return Client{nil, cfg}
}

func (c *Client) Connect() error {
	cl, err := qdrant.NewClient(&qdrant.Config{
		Host:                   c.cfg.Host,
		Port:                   c.cfg.Port,
		APIKey:                 c.cfg.API_KEY,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create a new Qdrant client: %w", err)
	}
	c.client = cl
	return nil
}

func (c Client) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection to Qdrant: %w", err)
	}
	return nil
}

func (c Client) CreateCollection(ctx context.Context, collName string) error {
	err := c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(c.cfg.EmbeddingSize),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	return err
}

func (c Client) Upsert(ctx context.Context, collName string, points []storage.Point) error {

	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i := range points {
		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(uint64(i)),
			Vectors: qdrant.NewVectors(points[i].Embedding...),
			Payload: qdrant.NewValueMap(points[i].Payload),
		}
	}

	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collName,
		Points:         qdrantPoints,
	})
	return err
}

func (c Client) Query(ctx context.Context, name string, embedding embed.Embedding) ([]*qdrant.ScoredPoint, error) {
	score, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: name,
		Query:          qdrant.NewQuery(embedding...),
	})
	return score, err
}
