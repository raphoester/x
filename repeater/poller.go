package repeater

import (
	"context"
	"time"

	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

type Config struct {
	Interval    time.Duration `yaml:"interval"`
	WaitOnStart bool          `yaml:"wait_on_start"`

	// If true, the repeater will adapt to slow consumers by dropping ticks.
	// If false, the repeater will just wait for the interval to pass after the receiver is done processing.
	DynamicInterval bool `yaml:"dynamic_interval"`
}

func (c *Config) ResetToDefault() {
	c.Interval = 1 * time.Minute
	c.WaitOnStart = true
}

func New(
	config Config,
	logger xlog.Logger,
	fn func(ctx context.Context) error,
) *Repeater {
	return &Repeater{
		interval: config.Interval,
		fn:       fn,
		logger:   logger,
	}
}

type Repeater struct {
	interval      time.Duration
	waitOnStart   bool
	fixedInterval bool
	logger        xlog.Logger
	fn            func(ctx context.Context) error
}

func (r *Repeater) Run(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(r.interval)
		first := true
		for {
			shouldRun := true
			if first && r.waitOnStart {
				shouldRun = false
				first = false
			}

			if shouldRun {
				if err := r.fn(ctx); err != nil {
					r.logger.Error("failed to poll", lf.Err(err))
				}
			}

			if r.fixedInterval {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			} else {
				select {
				case <-ctx.Done():
					return
				case <-time.After(r.interval):
				}
			}
		}
	}()
}
