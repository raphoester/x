package xmongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DBConfig struct {
	DSN          string `yaml:"dsn"`
	DatabaseName string `yaml:"database_name"`
}

func (c *DBConfig) ResetToDefault() {
	c.DSN = "mongodb://localhost:27017"
	c.DatabaseName = "default"
}

type ClientConfig struct {
	DSN string `yaml:"dsn"`
}

func (c *ClientConfig) ResetToDefault() {
	c.DSN = "mongodb://localhost:27017"
}

func NewClient(ctx context.Context, config ClientConfig) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.DSN))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo from DSN %q: %w", config.DSN, err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping mongo from DSN %q: %w", config.DSN, err)
	}

	return client, nil
}

func NewDatabase(ctx context.Context, config DBConfig) (*mongo.Database, error) {
	client, err := NewClient(ctx, ClientConfig{DSN: config.DSN})
	if err != nil {
		return nil, fmt.Errorf("failed to create client for database: %w", err)
	}

	return client.Database(config.DatabaseName), nil
}

func NopTX(ctx context.Context, fn func(ctx context.Context) (interface{}, error), _ ...*options.TransactionOptions) (interface{}, error) {
	return fn(ctx)
}
