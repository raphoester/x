package xevents

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/x/repeater"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

type PollerConfig struct {
	Repeater repeater.Config `yaml:"repeater"`
}

func (c *PollerConfig) ResetToDefault() {
	c.Repeater.ResetToDefault()
	c.Repeater.Interval = 5 * time.Second
}

func NewPoller(
	config PollerConfig,
	storage OutboxStorage,
	publisher Publisher,
	logger xlog.Logger,
) *Poller {
	p := &Poller{
		storage:   storage,
		publisher: publisher,
		logger:    logger,
	}
	rpt := repeater.New(config.Repeater, logger, p.poll)
	p.repeater = rpt

	return p
}

func (p *Poller) Run(ctx context.Context) error {
	p.repeater.Run(ctx)
	return nil
}

// Poller is a simple object that polls for pending events in the outbox storage and publishes them.
type Poller struct {
	storage   OutboxStorage
	publisher Publisher
	logger    xlog.Logger
	interval  time.Duration
	repeater  *repeater.Repeater
}

type OutboxStorage interface {
	GetPending(context.Context) ([]*Event, error)
	MarkAsPublished(ctx context.Context, id string) error
	Save(ctx context.Context, events ...*Event) error
}

func (p *Poller) poll(ctx context.Context) error {
	events, err := p.storage.GetPending(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get pending events: %w", err)
	}

	ctx = context.Background()
	for _, event := range events {
		p.logger.Debug("publishing event",
			lf.String("event_id", event.Data().ID),
			lf.String("event_topic", event.Data().Topic),
		)

		if err := p.publisher.Publish(ctx, event); err != nil {
			p.logger.Warning("failed to publish event", lf.Err(err))
			continue
		}

		if err := p.storage.MarkAsPublished(ctx, event.Data().ID); err != nil {
			p.logger.Error("failed to mark event as published", lf.Err(err))
			continue
		}
	}

	return nil
}
