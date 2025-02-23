package rabbitmq_broker

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/rabbitmq/amqp091-go"
	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xid"
	"github.com/raphoester/x/xrabbitmq"
)

func New(client *xrabbitmq.Client) (*Broker, error) {
	return &Broker{
		rabbitMQ:       client,
		idGenerator:    xid.RandomGenerator{},
		consumersCount: runtime.GOMAXPROCS(0),
	}, nil
}

type Broker struct {
	rabbitMQ       *xrabbitmq.Client
	idGenerator    xid.Generator
	consumersCount int
}

func (b *Broker) Publish(ctx context.Context, event xevents.Event) error {
	marshaledPayload, err := event.MarshalPayload()
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	if err := b.rabbitMQ.Publish(ctx, xrabbitmq.Payload{
		Topic:       event.Data().Topic,
		ContentType: "application/json",
		MessageID:   event.Data().ID,
		Timestamp:   event.Data().CreatedAt,
		Body:        marshaledPayload,
	}); err != nil {
		return fmt.Errorf("failed to push event: %w", err)
	}

	return nil
}

type HandlerPair struct {
	Topic   string
	Handler Handler
}

type Handler func(ctx context.Context, event xevents.Event) error

func (b *Broker) Listen(
	ctx context.Context,
	routingKeys []string,
	pairs ...HandlerPair,
) (chan struct{}, error) {
	if len(pairs) == 0 {
		return nil, errors.New("cannot listen without any handler pairs")
	}

	if len(routingKeys) == 0 {
		return nil, errors.New("cannot listen without any routing keys")
	}

	handlerMap := make(map[string]Handler)
	for _, pair := range pairs {
		handlerMap[pair.Topic] = pair.Handler
	}

	ready := make(chan struct{})

	go b.rabbitMQ.Stream(
		ctx,
		b.idGenerator.Generate(),
		routingKeys,
		ready,
		b.consumersCount,
		func(ctx context.Context, delivery amqp091.Delivery) error {
			handler, ok := handlerMap[delivery.RoutingKey]
			if !ok {
				return fmt.Errorf("no handler matching topic %q", delivery.RoutingKey)
			}

			event := xevents.Restore(
				delivery.MessageId,
				delivery.Timestamp,
				delivery.RoutingKey,
				delivery.Body,
			)

			if err := handler(ctx, event); err != nil {
				return fmt.Errorf("handler returned an error: %w", err)
			}

			return nil
		},
	)

	return ready, nil
}
