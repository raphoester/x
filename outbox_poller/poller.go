package outbox_poller

import (
	"context"
	"time"

	"github.com/raphoester/x/xevents"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

type Config struct {
	Interval time.Duration `yaml:"interval"`
}

func (c *Config) ResetToDefault() {
	c.Interval = 5 * time.Second
}

func New(
	storage OutboxStorage,
	publisher xevents.Publisher,
	logger xlog.Logger,
	config Config,
) *Poller {
	return &Poller{
		storage:   storage,
		publisher: publisher,
		logger:    logger,
		interval:  config.Interval,
	}
}

type Poller struct {
	storage   OutboxStorage
	publisher xevents.Publisher
	logger    xlog.Logger
	interval  time.Duration
	exit      chan struct{}
}

type OutboxStorage interface {
	GetPendingEvents(context.Context) ([]xevents.Event, error)
	MarkAsPublished(ctx context.Context, id string) error
}

func (p *Poller) Poll() {
	p.logger.Debug("polling for pending events to be sent")
	events, err := p.storage.GetPendingEvents(context.Background())
	if err != nil {
		p.logger.Error("failed to get pending events", lf.Err(err))
		return
	}

	ctx := context.Background()
	for _, event := range events {
		p.logger.Debug("publishing event",
			lf.String("event_id", event.Data().ID),
			lf.String("event_topic", event.Data().Topic),
		)

		if err := p.publisher.Publish(ctx, event); err != nil {
			p.logger.Error("failed to publish event", lf.Err(err))
			continue
		}

		if err := p.storage.MarkAsPublished(ctx, event.Data().ID); err != nil {
			p.logger.Error("failed to mark event as published", lf.Err(err))
			continue
		}
	}
}

func (p *Poller) Run() {
	ticker := time.NewTicker(p.interval)
	for {
		select {
		case <-ticker.C:
			p.Poll()
		case <-p.exit:
			ticker.Stop()
			return
		}
	}
}

func (p *Poller) Exit() {
	close(p.exit)
}
