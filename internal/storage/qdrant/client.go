package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"mail-assistant/internal/client/embed"
	"mail-assistant/internal/client/mail"
	"mail-assistant/internal/config"
	"mail-assistant/internal/storage"

	"github.com/qdrant/go-client/qdrant"
)

type Client struct {
	client *qdrant.Client
	config config.Qdrant
}

func New(config config.Qdrant) (Client, error) {
	qdrant, err := qdrant.NewClient(&qdrant.Config{
		Host:                   config.Host,
		Port:                   config.Port,
		APIKey:                 config.ApiKey,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return Client{}, fmt.Errorf("create a new Qdrant client: %w", err)
	}
	return Client{qdrant, config}, nil
}

func (c Client) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("close connection to Qdrant: %w", err)
	}
	return nil
}

func (c Client) Insert(ctx context.Context, collName string, points []storage.Point) error {
	if len(points) > 0 && len(points[0].Embedding) != c.config.EmbeddingSize {
		return fmt.Errorf("embedding size does`t match the collection dimension")
	}
	if len(points) == 0 {
		return nil
	}
	
	qdrantPoints := make([]*qdrant.PointStruct, len(points))

	for i := range points {
		raw, err := json.Marshal(points[i].Payload)
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
		UpdateMode:     qdrant.UpdateMode_InsertOnly.Enum(),
	})
	if err != nil {
		return fmt.Errorf("upsert query: %w", err)
	}
	return nil
}

func (c Client) Search(ctx context.Context, collName string, embedding embed.Embedding) ([]storage.ScoredPoint, error) {
	if len(embedding) != c.config.EmbeddingSize {
		return nil, fmt.Errorf("embedding size does`t match the collection dimension")
	}

	score, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: collName,
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
	exist, err := c.client.CollectionExists(ctx, collName)
	if err != nil {
		return fmt.Errorf("check collection existence: %w", err)
	}
	if exist {
		return nil
	}

	err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(c.config.EmbeddingSize),
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}
	return nil
}
