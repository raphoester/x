package xrabbitmq

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/raphoester/x/xlog"
	"github.com/raphoester/x/xlog/lf"
)

func NewConnection(
	url string,
	logger xlog.Logger,
) *Connection {
	return &Connection{
		url:    url,
		logger: logger,
	}
}

type Connection struct {
	url    string
	conn   *amqp091.Connection
	mutex  sync.Mutex
	logger xlog.Logger
}

// Connect establishes the connection to RabbitMQ
func (c *Connection) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil && !c.conn.IsClosed() {
		c.logger.Debug("rabbitmq connection already established")
		return nil
	}

	conn, err := amqp091.DialConfig(c.url, amqp091.Config{
		Heartbeat: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to dial server: %w", err)
	}
	c.conn = conn

	go c.monitorConnection()
	return nil
}

// monitorConnection watches for connection closure and reconnects
func (c *Connection) monitorConnection() {
	errChan := make(chan *amqp091.Error)
	c.conn.NotifyClose(errChan)
	err := <-errChan
	if err != nil {
		c.logger.Info("rabbitmq connection closed", lf.Err(err))
		c.mutex.Lock()
		c.conn = nil
		c.mutex.Unlock()
		// Attempt to reconnect
		for {
			time.Sleep(5 * time.Second)
			if err := c.Connect(); err == nil {
				c.logger.Info("reconnected to rabbitmq")
				break
			}

			c.logger.Warning("failed to reconnect to rabbitmq", lf.String("url", c.url), lf.Err(err))
		}
	}
}

// GetChannel returns a new channel from the connection
func (c *Connection) GetChannel() (*amqp091.Channel, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.conn == nil || c.conn.IsClosed() {
		return nil, errors.New("connection is closed")
	}
	return c.conn.Channel()
}
