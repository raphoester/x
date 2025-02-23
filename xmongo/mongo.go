package xmongo

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewConfigFromEnv() (Config, error) {
	dsn := os.Getenv("MONGODB_URI")
	if dsn == "" {
		return Config{}, fmt.Errorf("MONGODB_URI is empty")
	}

	return Config{DSN: dsn}, nil
}

type Config struct {
	DSN          string `yaml:"dsn"`
	DatabaseName string `yaml:"database_name"`
}

func (c *Config) ResetToDefault() {
	c.DSN = "mongodb://localhost:27017"
	c.DatabaseName = "default"
}

func NewClient(ctx context.Context, config Config) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(config.DSN))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping mongo: %w", err)
	}

	return client, nil
}

func NewDatabase(ctx context.Context, config Config) (*mongo.Database, error) {
	client, err := NewClient(ctx, config)
	if err != nil {
		return nil, err
	}

	return client.Database(config.DatabaseName), nil
}

func NopTX(ctx context.Context, fn func(ctx context.Context) (interface{}, error), _ ...*options.TransactionOptions) (interface{}, error) {
	return fn(ctx)
}
