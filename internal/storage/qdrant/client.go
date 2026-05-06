package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"mail-assistant/internal/config"
	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/storage"

	"github.com/qdrant/go-client/qdrant"
)

type Client struct {
	client *qdrant.Client
	cfg    *config.Qdrant
}

func New(cfg *config.Qdrant) (*Client, error) {
	cl, err := qdrant.NewClient(&qdrant.Config{
		Host:                   cfg.Host,
		Port:                   cfg.Port,
		APIKey:                 cfg.API_KEY,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return nil, fmt.Errorf("create a new Qdrant client: %w", err)
	}
	return &Client{cl, cfg}, nil
}

func (c Client) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("close connection to Qdrant: %w", err)
	}
	return nil
}

func (c Client) Upsert(ctx context.Context, collName string, points []storage.Point) error {
	if len(points) > 0 && len(points[0].Embedding) != c.cfg.EmbeddingSize {
		return fmt.Errorf("embedding size does`t match the collection dimension")
	}
	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i := range points {
		raw, err := json.Marshal(*points[i].Payload)
		if err != nil {
			return fmt.Errorf("marshal point payload: %w", err)
		}

		qdrantPoints[i] = &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(uint64(i)),
			Vectors: qdrant.NewVectors(points[i].Embedding...),
			Payload: map[string]*qdrant.Value{
				"raw_payload": qdrant.NewValueString(string(raw)),
			},
		}
	}

	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collName,
		Points:         qdrantPoints,
	})
	if err != nil {
		return fmt.Errorf("upsert query: %w", err)
	}
	return nil
}

func (c Client) Search(ctx context.Context, name string, embedding embed.Embedding) ([]storage.ScoredPoint, error) {
	if len(embedding) != c.cfg.EmbeddingSize {
		return nil, fmt.Errorf("embedding size does`t match the collection dimension")
	}

	score, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: name,
		Query:          qdrant.NewQuery(embedding...),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}

	res := make([]storage.ScoredPoint, len(score))
	for i, point := range score {
		res[i].Score = point.Score
		res[i].Payload = &mail.Letter{}
		if err = json.Unmarshal([]byte(point.Payload["raw_payload"].GetStringValue()), res[i].Payload); err != nil {
			return nil, fmt.Errorf("unmarshal point payload: %w", err)
		}
	}
	return res, nil
}

func (c Client) CreateCollection(ctx context.Context, collName string) error {
	err := c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(c.cfg.EmbeddingSize),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	return nil
}

func (c Client) DeleteCollection(ctx context.Context, collName string) error {
	err := c.client.DeleteCollection(ctx, collName)
	if err != nil {
		return fmt.Errorf("delete collection: %w", err)
	}
	return nil
}
