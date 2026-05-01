package qdrant

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

type Client struct {
	client *qdrant.Client
}

func (c *Client) Connect(host string, port int) error {
	cl, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   port,
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

func (c Client) CreateCollection(name string) error {
	err := c.client.CreateCollection(context.Background(), &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     4,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	return err
}
