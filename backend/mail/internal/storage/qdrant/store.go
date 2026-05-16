package qdrant

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"backend/mail/internal/client/embed"
	"backend/mail/internal/client/imap"
	"backend/mail/internal/config"
	"backend/mail/internal/storage"
)

type Store struct {
	client *qdrant.Client
	config config.Qdrant
}

func New(ctx context.Context, config config.Qdrant) (Store, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   config.Host,
		Port:                   config.Port,
		APIKey:                 config.ApiKey,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return Store{}, fmt.Errorf("create a new Qdrant client: %w", err)
	}

	store := Store{client: client, config: config}
	if err := store.createCollection(ctx, config.CollectionName); err != nil {
		return Store{}, fmt.Errorf("create collection: %w", err)
	}
	return store, nil
}

func (c Store) Close() error {
	err := c.client.Close()
	if err != nil {
		return fmt.Errorf("close connection to Qdrant: %w", err)
	}
	return nil
}

func (c Store) Insert(ctx context.Context, userID uuid.UUID, points []storage.Point) error {
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
			Id:      qdrant.NewIDUUID(uuid.NewString()),
			Vectors: qdrant.NewVectors(points[i].Embedding...),
			Payload: map[string]*qdrant.Value{
				"raw_payload": qdrant.NewValueString(string(raw)),
				"user_id":     qdrant.NewValueString(userID.String()),
			},
		}
	}

	_, err := c.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: c.config.CollectionName,
		Points:         qdrantPoints,
		UpdateMode:     qdrant.UpdateMode_InsertOnly.Enum(),
	})
	if err != nil {
		return fmt.Errorf("upsert query: %w", err)
	}
	return nil
}

func (c Store) Search(ctx context.Context, userID uuid.UUID, embedding embed.Embedding) ([]storage.ScoredPoint, error) {
	if len(embedding) != c.config.EmbeddingSize {
		return nil, fmt.Errorf("embedding size does`t match the collection dimension")
	}

	pointsLimit := c.config.SearchPointsLimit

	scored, err := c.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: c.config.CollectionName,
		Query:          qdrant.NewQuery(embedding...),
		WithPayload:    qdrant.NewWithPayload(true),
		Filter: &qdrant.Filter{
			Must: []*qdrant.Condition{{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "user_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: userID.String(),
							},
						},
					},
				},
			}},
		},
		Limit: &pointsLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}

	res := make([]storage.ScoredPoint, len(scored))
	for i, point := range scored {
		res[i].Score = point.Score
		res[i].Payload = &imap.Letter{}
		if err = json.Unmarshal([]byte(point.Payload["raw_payload"].GetStringValue()), res[i].Payload); err != nil {
			return nil, fmt.Errorf("unmarshal point payload: %w", err)
		}
	}
	return res, nil
}

func (c Store) createCollection(ctx context.Context, collName string) error {
	exist, err := c.client.CollectionExists(ctx, collName)
	if err != nil {
		return fmt.Errorf("check collection existence: %w", err)
	}
	if exist {
		return nil
	}

	storeOnDisk := true

	err = c.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: collName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     uint64(c.config.EmbeddingSize),
			Distance: qdrant.Distance_Cosine,
		}),
		OnDiskPayload: &storeOnDisk,
	})
	if err != nil {
		return fmt.Errorf("create collection: %w", err)
	}

	if err = c.createIndex(ctx, collName); err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	return nil
}

func (c Store) createIndex(ctx context.Context, collName string) error {
	_, err := c.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
		CollectionName: collName,
		FieldName:      "user_id",
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
	})
	if err != nil {
		return fmt.Errorf("create field index: %w", err)
	}
	return nil
}
