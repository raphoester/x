package mongo_helpers

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Snapshot[A any] interface {
	Restore() (*A, error)
}

func FindOne[S Snapshot[A], A any](ctx context.Context, collection *mongo.Collection, filter bson.M) (*A, error) {
	var snapshot S
	res := collection.FindOne(ctx, filter)
	if err := res.Err(); err != nil {
		return nil, MapErr(err)
	}

	if err := res.Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode snapshot: %w", err)
	}

	aggregate, err := snapshot.Restore()
	if err != nil {
		return nil, fmt.Errorf("failed to restore aggregate: %w", err)
	}

	return aggregate, nil
}

func FindMany[S Snapshot[A], A any](ctx context.Context, collection *mongo.Collection, filter bson.M) ([]*A, error) {
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshots: %w", err)
	}

	var snapshots []S
	if err := cursor.All(ctx, &snapshots); err != nil {
		return nil, fmt.Errorf("failed to decode snapshots: %w", err)
	}

	var aggregates []*A
	for _, snapshot := range snapshots {
		aggregate, err := snapshot.Restore()
		if err != nil {
			return nil, fmt.Errorf("failed to restore aggregate: %w", err)
		}

		aggregates = append(aggregates, aggregate)
	}

	return aggregates, nil
}
