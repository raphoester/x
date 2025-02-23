package xevents

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/raphoester/x/xid"
	"github.com/raphoester/x/xtime"
)

func New(
	timeProvider xtime.Provider,
	idGenerator xid.Generator,
	p Payload,
) Event {
	return Event{
		content: EventData{
			ID:        idGenerator.Generate(),
			CreatedAt: timeProvider.Now(),
			Topic:     p.Topic(),
			Payload:   p,
		},
	}
}

type Event struct {
	content EventData
}

func (e Event) ArbitraryStringProp(key string) (string, error) {
	rcv := map[string]interface{}{}
	if err := e.UnmarshalPayload(&rcv); err != nil {
		return "", fmt.Errorf("failed unmarshalling payload: %w", err)

	}

	value, ok := rcv[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in payload", key)
	}

	return fmt.Sprint(value), nil
}

// Restore offers a way to recreate an Event from its raw data.
//
// Payload is of any type to allow raw-feeding []bytes or any other type that doesn't necessarily implement Payload.
// The only constraint is that it should be json-unmarshal-able to the final concrete struct type
// that is passed to UnmarshalPayload.
func Restore(
	id string,
	createdAt time.Time,
	topic string,
	payload any,
) Event {
	return Event{
		content: EventData{
			ID:        id,
			CreatedAt: createdAt,
			Topic:     topic,
			Payload:   payload,
		},
	}
}

func (e Event) Data() EventData {
	return e.content
}

type EventData struct {
	ID        string
	CreatedAt time.Time
	Topic     string
	Payload   any
}

type Payload interface {
	Topic() string
	IsValid() bool
}

func (e Event) MarshalPayload() ([]byte, error) {
	return json.Marshal(e.content.Payload)
}

func (e Event) UnmarshalPayload(to any) error {
	b, ok := e.content.Payload.([]byte)
	if ok {
		if err := json.Unmarshal(b, &to); err != nil {
			return fmt.Errorf("failed unmarshalling payload: %w", err)
		}

		return nil
	}

	// if the payload is not a slice of bytes, we assume it's already under the form of a struct
	//
	// we then need to marshal it to []byte to be able to unmarshal it to the target type
	b, err := json.Marshal(e.content.Payload)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &to)
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
