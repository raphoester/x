package xdockertest

import (
	"context"
	"fmt"
	"time"

	"github.com/ory/dockertest"
	"github.com/raphoester/x/xmongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Mongo struct {
	Client    *mongo.Client
	container *dockertest.Resource
	pool      *dockertest.Pool
}

func (m *Mongo) Destroy() error {
	err := m.pool.Purge(m.container)
	return err
}

func (m *Mongo) Clean() error {
	dbs, err := m.Client.ListDatabaseNames(context.Background(), bson.M{
		"name": bson.M{"$nin": []string{"admin", "config", "local"}},
	})

	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	for _, db := range dbs {
		if err := m.Client.Database(db).Drop(context.Background()); err != nil {
			return fmt.Errorf("failed to drop database %s: %w", db, err)
		}
	}

	return nil
}

func NewMongo() (*Mongo, error) {
	pool, err := newPool()
	if err != nil {
		return nil, err
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mongo",
		Tag:        "8.0.0-noble",
		Cmd: []string{
			"--replSet",
			"rs0",
			"--bind_ip_all",
			"--port",
			"27017",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to run mongo container: %w", err)
	}

	port := container.GetPort("27017/tcp")
	uri := fmt.Sprintf("mongodb://localhost:%s/?directConnection=true&serverSelectionTimeoutMS=2000", port)

	var dbClient *mongo.Client
	if err := pool.Retry(func() error {
		dbClient, err = xmongo.NewClient(context.Background(), xmongo.ClientConfig{
			DSN: uri,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to mongo: %w", err)
		}

		// this is needed for the replicaset mode to work
		adminDB := dbClient.Database("admin")
		initCmd := bson.D{
			{"replSetInitiate", bson.M{
				"_id": "rs0",
				"members": bson.A{
					bson.M{
						"_id":  0,
						"host": "localhost:27017",
					},
				},
			}},
		}

		if err := adminDB.RunCommand(context.Background(), initCmd).Err(); err != nil {
			return fmt.Errorf("failed to initialize replicaset: %w", err)
		}

		time.Sleep(1 * time.Second) // wait for replicaset to initialize

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to connect to mongo: %w", err)
	}

	return &Mongo{
		Client:    dbClient,
		container: container,
		pool:      pool,
	}, nil
}
