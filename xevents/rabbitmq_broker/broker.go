package rabbitmq_broker

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/rabbitmq/amqp091-go"
	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
	"github.com/raphoester/x/xrabbitmq"
)

func New(client *xrabbitmq.Client, logger xlog.Logger) (*Broker, error) {
	return &Broker{
		rabbitMQ:       client,
		logger:         logger,
		consumersCount: runtime.GOMAXPROCS(0),
	}, nil
}

type Broker struct {
	rabbitMQ       *xrabbitmq.Client
	logger         xlog.Logger
	consumersCount int
}

func (b *Broker) Publish(ctx context.Context, event *xevents.Event) error {
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

func (b *Broker) Listen(ctx context.Context, identifier string, routingKeys []string, pairs ...xevents.HandlerPair) error {
	if len(pairs) == 0 {
		return errors.New("cannot listen without any handler pairs")
	}

	if len(routingKeys) == 0 {
		return errors.New("cannot listen without any routing keys")
	}

	handlerMap := make(map[string]xevents.Handler)
	for _, pair := range pairs {
		handlerMap[pair.Topic] = pair.Handler
	}

	ready := make(chan struct{})

	go b.rabbitMQ.Stream(
		ctx,
		identifier,
		routingKeys,
		ready,
		b.consumersCount,
		func(ctx context.Context, delivery amqp091.Delivery) error {
			handler, ok := handlerMap[delivery.RoutingKey]
			if !ok {
				b.logger.Info(
					"received unprocessable topic, dropping",
					lf.String("topic", delivery.RoutingKey),
					lf.String("message_id", delivery.MessageId),
				)
				return nil
			}

			event := xevents.Restore(
				delivery.MessageId,
				delivery.Timestamp.UTC(),
				delivery.RoutingKey,
				delivery.Body,
			)

			if err := handler(ctx, event); err != nil {
				return fmt.Errorf("handler returned an error: %w", err)
			}

			return nil
		},
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ready:
		return nil
	}
}
