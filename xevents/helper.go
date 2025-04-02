package xevents

import (
	"context"
	"errors"
	"fmt"
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
func UnmarshalHelper[P Payload](fn func(ctx context.Context, event *Event, payload P) error) func(ctx context.Context, event *Event) error {
	return func(ctx context.Context, event *Event) error {
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
