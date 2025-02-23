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
	subscribers map[string][]subscriber
	quit        chan struct{}
	closed      bool
	logger      xlog.Logger
}

func New(logger xlog.Logger) *Broker {
	return &Broker{
		subscribers: make(map[string][]subscriber),
		quit:        make(chan struct{}),
		logger:      logger,
	}
}

type subscriber struct {
	channel chan xevents.Event
}

func (b *Broker) Publish(_ context.Context, event xevents.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.logger.Debug("publishing event",
		lf.String("topic", event.Data().Topic),
		lf.Int("subscriber_count", len(b.subscribers[event.Data().Topic])),
	)

	for i, sub := range b.subscribers[event.Data().Topic] {
		b.logger.Debug("sending event",
			lf.String("topic", event.Data().Topic),
			lf.Int("subscriber_index", i),
		)
		go func() { sub.channel <- event }() // avoid blocking all receivers if one is slow
	}
	return nil
}

func (b *Broker) Subscribe(topic string) <-chan xevents.Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	ch := make(chan xevents.Event)
	b.subscribers[topic] = append(
		b.subscribers[topic],
		subscriber{
			channel: ch,
		},
	)

	return ch
}

func (b *Broker) Unsubscribe(topic string, ch <-chan xevents.Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return
	}

	index := -1
	for i, sub := range b.subscribers[topic] {
		if sub.channel == ch {
			index = i
			break
		}
	}

	if index != -1 {
		close(b.subscribers[topic][index].channel)
		b.subscribers[topic] = append(b.subscribers[topic][:index], b.subscribers[topic][index+1:]...)
	}
}

func (b *Broker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	close(b.quit)

	for _, ch := range b.subscribers {
		for _, sub := range ch {
			close(sub.channel)
		}
	}

	return nil
}

func (b *Broker) Done() <-chan struct{} {
	return b.quit
}
