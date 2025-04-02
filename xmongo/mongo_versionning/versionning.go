package mongo_versionning

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/x/xerrs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Aggregate[S any] interface {
	ID() string
	Loaded() int
	Modified() bool
	TakeSnapshot() S
}

func Upsert[S any, A Aggregate[S]](ctx context.Context, collection *mongo.Collection, aggregate A) error {
	if !aggregate.Modified() {
		return nil
	}
	_, err := collection.UpdateOne(ctx,
		bson.M{
			"_id":     aggregate.ID(),
			"version": aggregate.Loaded(),
		}, bson.M{
			"$set": aggregate.TakeSnapshot(),
		}, options.Update().SetUpsert(true),
	)

	if mongo.IsDuplicateKeyError(err) {
		return xerrs.ErrConflict
	}

	if err != nil {
		return fmt.Errorf("failed to upsert aggregate: %w", err)
	}

	return nil
}

func AssertVersion(ctx context.Context, collection *mongo.Collection, wantedVersion int, id string) error {
	var snapshot struct {
		Version int `bson:"version"`
	}
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&snapshot)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return xerrs.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	if snapshot.Version != wantedVersion {
		return xerrs.ErrConflict
	}

	return nil
}
