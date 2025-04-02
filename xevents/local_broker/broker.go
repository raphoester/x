package local_broker

import (
	"context"
	"sync"

	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

type Broker struct {
	mu          sync.Mutex
	subscribers []subscriber
	quit        chan struct{}
	closed      bool
	logger      xlog.Logger
}

func New(logger xlog.Logger) *Broker {
	return &Broker{
		subscribers: make([]subscriber, 0),
		quit:        make(chan struct{}),
		logger:      logger,
	}
}

type subscriber struct {
	ctx      context.Context
	handlers map[string]xevents.Handler
}

func (b *Broker) Publish(_ context.Context, event *xevents.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.logger.Debug("publishing event",
		lf.String("topic", event.Data().Topic),
	)

	for i, sub := range b.subscribers {
		handler, ok := sub.handlers[event.Data().Topic]
		if !ok {
			continue
		}

		b.logger.Debug("sending event",
			lf.String("topic", event.Data().Topic),
			lf.Int("subscriber_index", i),
		)

		go func() {
			err := handler(sub.ctx, event)
			if err != nil {
				b.logger.Error("failed to handle event",
					lf.String("topic", event.Data().Topic),
					lf.Err(err),
				)
			}
		}()
	}
	return nil
}

func (b *Broker) Listen(ctx context.Context, identifier string, routingKeys []string, pairs ...xevents.HandlerPair) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	if len(pairs) == 0 {
		return nil
	}

	if len(routingKeys) == 0 {
		return nil
	}

	handlerMap := make(map[string]xevents.Handler)
	for _, pair := range pairs {
		handlerMap[pair.Topic] = pair.Handler
	}

	sub := subscriber{
		ctx:      ctx,
		handlers: handlerMap,
	}

	b.subscribers = append(b.subscribers, sub)
	b.logger.Debug("subscribed to topics",
		lf.String("identifier", identifier),
		lf.Strings("routingKeys", routingKeys),
	)

	return nil
}

func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	close(b.quit)

	return nil
}

func (b *Broker) Done() <-chan struct{} {
	return b.quit
}
