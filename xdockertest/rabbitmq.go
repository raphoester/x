package xdockertest

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ory/dockertest"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xrabbitmq"
)

type RabbitMQ struct {
	RabbitMQ  *xrabbitmq.Client
	container *dockertest.Resource
	pool      *dockertest.Pool
	logger    xlog.Logger
}

func (r *RabbitMQ) Destroy() error {
	return r.pool.Purge(r.container)
}

func (r *RabbitMQ) Clean() error {
	return nil
}

func NewRabbitMQ(
	logger xlog.Logger,
) (*RabbitMQ, error) {
	pool, err := newPool()
	if err != nil {
		return nil, err
	}

	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "rabbitmq",
		Tag:        "3.9.7-management",
		Env: []string{
			"RABBITMQ_DEFAULT_USER=guest",
			"RABBITMQ_DEFAULT_PASS=guest",
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to run rabbitmq container: %w", err)
	}

	port := container.GetPort("5672/tcp")
	url := fmt.Sprintf("amqp://guest:guest@localhost:%s", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		os.Interrupt,
		os.Kill,
	)

	var client *xrabbitmq.Client
	if err := pool.Retry(func() error {
		client, err = xrabbitmq.NewClient(
			logger,
			xrabbitmq.Config{
				URL:          url,
				ExchangeName: "x-dockertest-exchange-name",
				RetryDelay:   5 * time.Second,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to connect to rabbitmq: %w", err)
		}

		time.Sleep(5 * time.Second)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed creating rabbitmq client: %w", err)
	}

	return &RabbitMQ{
		RabbitMQ:  client,
		container: container,
		pool:      pool,
		logger:    logger,
	}, nil

}
