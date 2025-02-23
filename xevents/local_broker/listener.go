package local_broker

import (
	"context"

	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

type Handler struct {
	Topic  string
	Handle func(ctx context.Context, event xevents.Event) error
}

func NewListener(
	broker *Broker,
	logger xlog.Logger,
	handlers ...Handler,
) *Listener {
	pairs := make(map[string]handlerPair)
	for _, h := range handlers {
		pairs[h.Topic] = handlerPair{
			handler: h.Handle,
			ch:      broker.Subscribe(h.Topic),
		}
	}

	return &Listener{
		pairs:  pairs,
		broker: broker,
		logger: logger,
	}
}

type handlerPair struct {
	handler func(ctx context.Context, event xevents.Event) error
	ch      <-chan xevents.Event
}

// Listener listens for events from the broker and handles them with the provided handlers.
//
// It is used for simulating topic based routing the same way as an actual broker would.
// Using it in production is not recommended.
type Listener struct {
	pairs  map[string]handlerPair
	broker *Broker
	logger xlog.Logger
}

func (s *Listener) Run() {
	for topic, pair := range s.pairs {
		go func(topic string, pair handlerPair) {
			for {
				select {
				case event := <-pair.ch:
					s.goHandle(pair.handler, event)
				case <-s.broker.Done():
					return
				}
			}
		}(topic, pair)
	}
}

func (s *Listener) Exit() {
	for topic, ch := range s.pairs {
		s.broker.Unsubscribe(topic, ch.ch)
	}
}

func (s *Listener) goHandle(fn func(ctx context.Context, event xevents.Event) error, event xevents.Event) {
	go func() {
		s.logger.Info("handling event",
			lf.String("event_id", event.Data().ID),
			lf.Time("event_created_at", event.Data().CreatedAt),
			lf.String("event_topic", event.Data().Topic),
		)
		ctx := context.Background()
		if err := fn(ctx, event); err != nil {
			s.logger.Error(
				"failed to handle event",
				lf.String("event_id", event.Data().ID),
				lf.Time("event_created_at", event.Data().CreatedAt),
				lf.String("event_topic", event.Data().Topic),
				lf.Err(err),
			)
		}
	}()
}
