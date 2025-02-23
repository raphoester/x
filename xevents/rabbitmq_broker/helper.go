package rabbitmq_broker

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/x/xevents"
)

// UnmarshalHelper is a utility function that allows the handler NOT to care about any of the unmarshalling logic.
//
// Example usage:
//
//	func exampleUsage() {
//		_ = (&Broker{}).Listen(context.Background(), nil, "someQueue", "somePattern", HandlerPair{
//			Topic: "someTopic",
//			Handler: UnmarshalHelper[xevents.ExamplePayload](
//				func(ctx context.Context, event xevents.Event, payload xevents.ExamplePayload) error {
//					/* perform treatment on payload or whatever */
//					_ = payload.Key
//					return nil
//				},
//			),
//		})
//	}
func UnmarshalHelper[P xevents.Payload](fn func(ctx context.Context, event xevents.Event, payload P) error) func(ctx context.Context, event xevents.Event) error {
	return func(ctx context.Context, event xevents.Event) error {
		var payload P
		if err := event.UnmarshalPayload(&payload); err != nil {
			return fmt.Errorf("failed unmarshaling payload to %T: %w", payload, err)
		}

		if !payload.IsValid() {
			return errors.New("unmarshalled payload is invalid")
		}

		return fn(ctx, event, payload)
	}
}
