package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/redis/go-redis/v9"
)

const (
	ConfigChannelPrefix = "config:"
	GlobalConfigChannel = "config:global"
)

// Client wraps Redis client with pub/sub functionality
type Client struct {
	rdb    *redis.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// Config holds Redis connection configuration
type Config struct {
	Address  string
	Password string
	DB       int
	Enabled  bool
}

// NewClient creates a new Redis client
func NewClient(config Config) (*Client, error) {
	if !config.Enabled {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		cancel()
		return nil, err
	}

	logger.Log.Info("Connected to Redis successfully")

	return &Client{
		rdb:    rdb,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.cancel()
	return c.rdb.Close()
}

// IsConnected checks if Redis is connected
func (c *Client) IsConnected() bool {
	if c == nil {
		return false
	}
	return c.rdb.Ping(c.ctx).Err() == nil
}

// PublishConfig publishes configuration change to Redis
func (c *Client) PublishConfig(config models.WorkerConfig, version string) error {
	if c == nil {
		return nil
	}

	message := ConfigMessage{
		Config:    config,
		Version:   version,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Publish to global config channel
	err = c.rdb.Publish(c.ctx, GlobalConfigChannel, data).Err()
	if err != nil {
		logger.Log.Errorf("Failed to publish config to Redis: %v", err)
		return err
	}

	logger.Log.Infof("Published config change to Redis channel: %s", GlobalConfigChannel)
	return nil
}

// SubscribeToConfig subscribes to configuration changes
func (c *Client) SubscribeToConfig() (<-chan ConfigMessage, error) {
	if c == nil {
		return nil, nil
	}

	pubsub := c.rdb.Subscribe(c.ctx, GlobalConfigChannel)
	ch := make(chan ConfigMessage, 10)

	go func() {
		defer close(ch)
		defer pubsub.Close()

		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				msg, err := pubsub.ReceiveMessage(c.ctx)
				if err != nil {
					if c.ctx.Err() != nil {
						return
					}
					logger.Log.Errorf("Error receiving Redis message: %v", err)
					time.Sleep(time.Second)
					continue
				}

				var configMsg ConfigMessage
				if err := json.Unmarshal([]byte(msg.Payload), &configMsg); err != nil {
					logger.Log.Errorf("Failed to unmarshal config message: %v", err)
					continue
				}

				logger.Log.Infof("Received config change from Redis: version %s", configMsg.Version)

				select {
				case ch <- configMsg:
				case <-c.ctx.Done():
					return
				default:
					// Channel is full, drop the message
					logger.Log.Warn("Config message channel is full, dropping message")
				}
			}
		}
	}()

	return ch, nil
}

// ConfigMessage represents a configuration message in Redis
type ConfigMessage struct {
	Config    models.WorkerConfig `json:"config"`
	Version   string              `json:"version"`
	Timestamp time.Time           `json:"timestamp"`
}

// GetConfigFromRedis retrieves the latest config from Redis (for backup)
func (c *Client) GetConfigFromRedis() (*ConfigMessage, error) {
	if c == nil {
		return nil, nil
	}

	data, err := c.rdb.Get(c.ctx, "latest_config").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No config stored
		}
		return nil, err
	}

	var configMsg ConfigMessage
	err = json.Unmarshal([]byte(data), &configMsg)
	if err != nil {
		return nil, err
	}

	return &configMsg, nil
}

// StoreConfigInRedis stores the latest config in Redis for backup
func (c *Client) StoreConfigInRedis(config models.WorkerConfig, version string) error {
	if c == nil {
		return nil
	}

	message := ConfigMessage{
		Config:    config,
		Version:   version,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Store with expiration (24 hours)
	return c.rdb.Set(c.ctx, "latest_config", data, 24*time.Hour).Err()
}