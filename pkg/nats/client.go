package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/doniyusdinar/config-management/pkg/logger"
)

// Config holds NATS configuration
type Config struct {
	URLs            []string      `json:"urls"`
	Username        string        `json:"username"`
	Password        string        `json:"password"`
	Token           string        `json:"token"`
	TLSEnabled      bool          `json:"tls_enabled"`
	MaxReconnect    int           `json:"max_reconnect"`
	ReconnectWait   time.Duration `json:"reconnect_wait"`
	ConnectionName  string        `json:"connection_name"`
	Subject         string        `json:"subject"`
	QueueGroup      string        `json:"queue_group"`
	Enabled         bool          `json:"enabled"`
}

// Client wraps NATS connection with additional functionality
type Client struct {
	conn   *nats.Conn
	config Config
	ctx    context.Context
	cancel context.CancelFunc
}

// NewClient creates a new NATS client
func NewClient(config Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Connect establishes connection to NATS server
func (c *Client) Connect() error {
	if !c.config.Enabled {
		return fmt.Errorf("NATS client is disabled")
	}

	opts := []nats.Option{
		nats.Name(c.config.ConnectionName),
		nats.MaxReconnects(c.config.MaxReconnect),
		nats.ReconnectWait(c.config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Log.Warnf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Log.Infof("NATS reconnected to %v", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			logger.Log.Warnf("NATS connection closed")
		}),
	}

	// Add authentication if provided
	if c.config.Token != "" {
		opts = append(opts, nats.Token(c.config.Token))
	} else if c.config.Username != "" && c.config.Password != "" {
		opts = append(opts, nats.UserInfo(c.config.Username, c.config.Password))
	}

	// Add TLS if enabled
	if c.config.TLSEnabled {
		opts = append(opts, nats.Secure())
	}

	var err error
	c.conn, err = nats.Connect(nats.DefaultURL, opts...)
	if len(c.config.URLs) > 0 {
		url := c.config.URLs[0]
		if len(c.config.URLs) > 1 {
			url = nats.DefaultURL
			for i, u := range c.config.URLs {
				if i == 0 {
					url = u
				} else {
					url += "," + u
				}
			}
		}
		c.conn, err = nats.Connect(url, opts...)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %v", err)
	}

	logger.Log.Infof("NATS client connected to %s", c.conn.ConnectedUrl())
	return nil
}

// Subscribe creates a subscription to a subject
func (c *Client) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("NATS client not connected")
	}
	return c.conn.Subscribe(subject, handler)
}

// QueueSubscribe creates a queue subscription to a subject
func (c *Client) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("NATS client not connected")
	}
	return c.conn.QueueSubscribe(subject, queue, handler)
}

// Publish publishes a message to a subject
func (c *Client) Publish(subject string, data []byte) error {
	if c.conn == nil {
		return fmt.Errorf("NATS client not connected")
	}
	return c.conn.Publish(subject, data)
}

// PublishRequest publishes a request and waits for a response
func (c *Client) PublishRequest(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("NATS client not connected")
	}
	return c.conn.Request(subject, data, timeout)
}

// Flush ensures all pending messages are sent
func (c *Client) Flush() error {
	if c.conn == nil {
		return fmt.Errorf("NATS client not connected")
	}
	return c.conn.Flush()
}

// Close closes the NATS connection
func (c *Client) Close() {
	c.cancel()
	if c.conn != nil {
		c.conn.Close()
		logger.Log.Info("NATS client connection closed")
	}
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Stats returns connection statistics
func (c *Client) Stats() nats.Statistics {
	if c.conn == nil {
		return nats.Statistics{}
	}
	return c.conn.Stats()
}

// HealthCheck performs a health check on the NATS connection
func (c *Client) HealthCheck() error {
	if c.conn == nil {
		return fmt.Errorf("NATS client not connected")
	}
	
	if !c.conn.IsConnected() {
		return fmt.Errorf("NATS connection is down")
	}
	
	// Test with a ping
	err := c.conn.Flush()
	if err != nil {
		return fmt.Errorf("NATS health check failed: %v", err)
	}
	
	return nil
}