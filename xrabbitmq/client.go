package xrabbitmq

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

type Config struct {
	URL          string        `yaml:"url"`
	ExchangeName string        `yaml:"exchange_name"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
}

func (c *Config) ResetToDefault() {
	c.URL = "amqp://guest:guest@localhost:5672/"
	threads := runtime.GOMAXPROCS(0)
	if numCPU := runtime.NumCPU(); numCPU > threads {
		threads = numCPU
	}
}

// NewClient creates a new RabbitMQ client
func NewClient(
	logger xlog.Logger,
	config Config,
) (*Client, error) {
	conn := NewConnection(config.URL, logger)
	if err := conn.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Declare the exchange
	ch, err := conn.GetChannel()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}
	defer func() { _ = ch.Close() }()

	err = ch.ExchangeDeclare(
		config.ExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &Client{
		connection: conn,
		exchange:   config.ExchangeName,
		logger:     logger,
		retryDelay: config.RetryDelay,
	}, nil
}

// Client is the RabbitMQ client that manages publishing and consuming
type Client struct {
	connection *Connection
	exchange   string

	publishChannel *amqp091.Channel
	publishMutex   sync.Mutex

	logger     xlog.Logger
	retryDelay time.Duration
}

type Payload struct {
	Topic       string
	ContentType string
	MessageID   string
	Timestamp   time.Time
	Body        []byte
}

func (c *Client) Publish(ctx context.Context, payload Payload) error {
	c.publishMutex.Lock()
	defer c.publishMutex.Unlock()

	for {
		if c.publishChannel == nil || c.publishChannel.IsClosed() {
			ch, err := c.connection.GetChannel()
			if err != nil {
				return fmt.Errorf("failed to get channel: %w", err)
			}
			c.publishChannel = ch
		}

		err := c.publishChannel.PublishWithContext(ctx,
			c.exchange,
			payload.Topic,
			false,
			false,
			amqp091.Publishing{
				ContentType:  payload.ContentType,
				MessageId:    payload.MessageID,
				Timestamp:    payload.Timestamp,
				Body:         payload.Body,
				Expiration:   "", // message doesn't expire
				DeliveryMode: 2,
			},
		)
		if err == nil {
			return nil
		}

		if errors.Is(err, amqp091.ErrClosed) {
			c.publishChannel = nil
			continue
		}

		return fmt.Errorf("failed to publish message: %w", err)
	}
}

// consumerLoop runs a single consumer, handling reconnects
func consumerLoop(
	connection *Connection,
	queueName string,
	routingKeys []string,
	exchange string,
	ready chan<- struct{},
	callback func(context.Context, amqp091.Delivery) error,
	stop <-chan struct{},
	retryDelay time.Duration,
	logger xlog.Logger,
	identifier int,
) {
	for {
		logger.Debug("consumer loop start", lf.Int("identifier", identifier))
		select {
		case <-stop:
			logger.Info("consumer loop stop", lf.Int("identifier", identifier))
			return
		default:
		}

		ch, err := connection.GetChannel()
		if err != nil {
			logger.Warning(
				"failed to get channel",
				lf.Int("identifier", identifier),
				lf.Err(err),
			)
			time.Sleep(retryDelay)
			continue
		}

		obtainMsgsChan := func() (<-chan amqp091.Delivery, error) {
			if _, err := ch.QueueDeclare(
				queueName,
				true,
				false,
				false,
				false,
				nil,
			); err != nil {
				return nil, fmt.Errorf("failed to declare queue: %w", err)
			}

			for _, routingKey := range routingKeys {
				// Bind queue to exchange
				if err = ch.QueueBind(
					queueName,
					routingKey,
					exchange,
					false,
					nil,
				); err != nil {
					return nil, fmt.Errorf("failed to bind queue on routing key %q: %w", routingKey, err)
				}
			}

			// Start consuming
			msgs, err := ch.Consume(
				queueName,
				"",
				false,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to consume: %w", err)
			}

			ready <- struct{}{}

			return msgs, nil
		}

		var msgs <-chan amqp091.Delivery
		for {
			msgs, err = obtainMsgsChan()
			if err == nil {
				break
			}

			logger.Warning("failed to obtain channel with messages", lf.Int("identifier", identifier), lf.Err(err))
			_ = ch.Close()
			time.Sleep(retryDelay)
		}

		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					// Channel closed
					_ = ch.Close()
					break
				}

				logger.Debug(
					"received delivery",
					lf.Int("identifier", identifier),
					lf.String("routing_key", msg.RoutingKey),
					lf.String("message_id", msg.MessageId),
				)

				handleMessageWithAck(msg, callback, logger)
			case <-stop:
				_ = ch.Close()
				return
			}
		}
	}
}

// Stream sets up N consumers on the specified queue with the given routing key
func (c *Client) Stream(
	ctx context.Context,
	queue string,
	routingKeys []string,
	ready chan<- struct{},
	numConsumers int,
	callback func(context.Context, amqp091.Delivery) error,
) {

	allReady := make([]chan struct{}, 0, numConsumers)
	allClose := make([]chan struct{}, 0, numConsumers)

	for i := range numConsumers {
		readyCh := make(chan struct{})
		closeCh := make(chan struct{})

		allReady = append(allReady, readyCh)
		allClose = append(allClose, closeCh)

		go consumerLoop(c.connection, queue, routingKeys, c.exchange, readyCh, callback, closeCh, c.retryDelay, c.logger, i)
	}

	for _, readyCh := range allReady {
		<-readyCh
	}

	// All consumers are ready
	close(ready)

	for {
		select {
		case <-ctx.Done():
			for _, closeCh := range allClose {
				close(closeCh)
			}
			return
		}
	}
}

// handleMessageWithAck wraps the callback to handle ACK/NACK
func handleMessageWithAck(
	msg amqp091.Delivery,
	callback func(context.Context, amqp091.Delivery) error,
	logger xlog.Logger,
) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(
				"panic occurred while treating delivery",
				lf.Any("panic", r),
			)
			// If callback panics, NACK the message
			err := msg.Nack(false, true)
			if err != nil {
				logger.Warning(
					"failed to nack message",
					lf.Err(err),
				)
			}
		}
	}()

	if err := callback(context.TODO(), msg); err != nil {
		logger.Warning(
			"failed to treat delivery",
			lf.Err(err),
		)
		if err := msg.Nack(false, true); err != nil {
			logger.Warning(
				"failed to nack message",
				lf.Err(err),
			)
		}
		return
	}

	if err := msg.Ack(false); err != nil {
		logger.Warning(
			"failed to ack message",
			lf.Err(err),
		)
	}
}
