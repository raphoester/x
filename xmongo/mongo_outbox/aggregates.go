package mongo_outbox

import (
	"context"
	"fmt"

	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xmongo"
	"github.com/raphoester/x/xmongo/mongo_versionning"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Aggregate[S any] interface {
	Collect() []*xevents.Event
	mongo_versionning.Aggregate[S]
}

func SaveAggregate[S any, EA Aggregate[S]](
	ctx context.Context,
	collection *mongo.Collection,
	aggregate EA,
) error {
	ev := aggregate.Collect()
	db := collection.Database()

	modified := aggregate.Modified()

	// use a transaction if the aggregate has events:
	// - if the aggregate is modified, need to update both in an atomic way
	// - if the aggregate is not modified, need to check for version conflicts before saving the events
	tx := xmongo.NopTX
	if len(ev) > 0 {
		tx = func(ctx context.Context, fn func(ctx2 context.Context) (interface{}, error), _ ...*options.TransactionOptions) (interface{}, error) {
			session, err := db.Client().StartSession()
			if err != nil {
				return nil, fmt.Errorf("failed to start session for transaction: %w", err)
			}
			defer session.EndSession(context.Background())
			return session.WithTransaction(ctx, func(ctx mongo.SessionContext) (interface{}, error) {
				return fn(ctx)
			})
		}
	}

	saveFn := func(ctx context.Context) (interface{}, error) {
		if err := mongo_versionning.Upsert(ctx, collection, aggregate); err != nil {
			return nil, fmt.Errorf("failed to upsert aggregate: %w", err)
		}

		if err := SaveEvents(ctx, db, ev); err != nil {
			return nil, fmt.Errorf("failed to save events: %w", err)
		}

		return nil, nil
	}

	if !modified && len(ev) > 0 { // do not allow the aggregate's events to be saved if the corresponding version is conflicting
		saveFn = func(ctx context.Context) (interface{}, error) {
			if err := SaveEvents(ctx, db, ev); err != nil {
				return nil, fmt.Errorf("failed to save events: %w", err)
			}

			// check if the version is conflicting
			if err := mongo_versionning.AssertVersion(ctx, collection, aggregate.Loaded(), aggregate.ID()); err != nil {
				return nil, fmt.Errorf("failed to assert version: %w", err)
			}

			return nil, nil
		}
	}

	if _, err := tx(ctx, saveFn); err != nil {
		return fmt.Errorf("failed to save aggregate with its events: %w", err)
	}

	return nil
}
