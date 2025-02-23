package mongo_outbox

import (
	"context"
	"fmt"

	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xmongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Aggregate interface {
	Collect() []xevents.Event
}

type SaveFunc[T Aggregate] func(context.Context, *mongo.Collection, T) error

func SaveAggregate[T Aggregate](
	ctx context.Context,
	collection *mongo.Collection,
	aggregate T,
	saveFunc SaveFunc[T],
) error {
	ev := aggregate.Collect()
	db := collection.Database()

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

	_, err := tx(ctx, func(ctx2 context.Context) (interface{}, error) {
		if len(ev) > 0 {
			if err := SaveEvents(ctx2, db, ev); err != nil {
				return nil, fmt.Errorf("failed to save events: %w", err)
			}
		}

		if err := saveFunc(ctx2, collection, aggregate); err != nil {
			return nil, fmt.Errorf("failed to save aggregate: %w", err)
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("failed to save aggregate with its events: %w", err)
	}

	return nil
}
