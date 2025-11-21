package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/doniyusdinar/config-management/agent/internal/worker"
	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/doniyusdinar/config-management/pkg/redis"
	natspkg "github.com/doniyusdinar/config-management/pkg/nats"
	"github.com/nats-io/nats.go"
)

// DistributionStrategy represents the config distribution strategy
type DistributionStrategy string

const (
	StrategyPoller DistributionStrategy = "POLLER"
	StrategyRedis  DistributionStrategy = "REDIS"
	StrategyNats   DistributionStrategy = "NATS"
	// Future strategies:
	// StrategyKafka  DistributionStrategy = "KAFKA"
)

// ConfigDistributor interface for different distribution strategies
type ConfigDistributor interface {
	Start(ctx context.Context) error
	Stop() error
	GetType() DistributionStrategy
}

// PollerDistributor implements HTTP polling strategy
type PollerDistributor struct {
	poller *Poller
}

func NewPollerDistributor(controllerURL, username, password string, workerMgr *worker.Manager, cacheFile string) *PollerDistributor {
	return &PollerDistributor{
		poller: NewPoller(controllerURL, username, password, workerMgr, cacheFile),
	}
}

func (pd *PollerDistributor) Start(ctx context.Context) error {
	logger.Log.Info("Starting HTTP polling distribution strategy")
	return pd.poller.Start(ctx)
}

func (pd *PollerDistributor) Stop() error {
	// Poller stops via context cancellation
	return nil
}

func (pd *PollerDistributor) GetType() DistributionStrategy {
	return StrategyPoller
}

// RedisDistributor implements Redis pub/sub strategy
type RedisDistributor struct {
	redisClient *redis.Client
	workerMgr   *worker.Manager
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	lastConfig  *models.WorkerConfig
	lastVersion string
}

func NewRedisDistributor(redisConfig redis.Config, workerMgr *worker.Manager) (*RedisDistributor, error) {
	redisClient, err := redis.NewClient(redisConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RedisDistributor{
		redisClient: redisClient,
		workerMgr:   workerMgr,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

func (rd *RedisDistributor) Start(ctx context.Context) error {
	logger.Log.Info("Starting Redis pub/sub distribution strategy")

	// Subscribe to Redis config changes
	configChan, err := rd.redisClient.SubscribeToConfig()
	if err != nil {
		return fmt.Errorf("failed to subscribe to Redis config: %w", err)
	}

	// Handle Redis messages
	go rd.handleRedisMessages(configChan)

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

func (rd *RedisDistributor) handleRedisMessages(configChan <-chan redis.ConfigMessage) {
	logger.Log.Info("Redis config subscriber started")

	for {
		select {
		case <-rd.ctx.Done():
			logger.Log.Info("Redis subscriber shutting down")
			return
		case configMsg, ok := <-configChan:
			if !ok {
				logger.Log.Warn("Redis config channel closed")
				return
			}

			rd.mu.Lock()
			// Check if this is a new version
			if configMsg.Version != rd.lastVersion {
				rd.lastConfig = &configMsg.Config
				rd.lastVersion = configMsg.Version

				logger.Log.Infof("Received new config from Redis: version %s", configMsg.Version)

				// Forward to worker
				if err := rd.workerMgr.ForwardConfig(configMsg.Config); err != nil {
					logger.Log.Errorf("Failed to forward Redis config to worker: %v", err)
				} else {
					logger.Log.Info("Successfully forwarded Redis config to worker")
				}
			}
			rd.mu.Unlock()
		}
	}
}

func (rd *RedisDistributor) Stop() error {
	logger.Log.Info("Stopping Redis distributor")
	rd.cancel()
	if rd.redisClient != nil {
		return rd.redisClient.Close()
	}
	return nil
}

func (rd *RedisDistributor) GetType() DistributionStrategy {
	return StrategyRedis
}

func (rd *RedisDistributor) GetLastConfig() *models.WorkerConfig {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.lastConfig
}

func (rd *RedisDistributor) GetLastVersion() string {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.lastVersion
}

// NatsDistributor implements NATS pub/sub strategy
type NatsDistributor struct {
	natsClient  *natspkg.Client
	workerMgr   *worker.Manager
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	lastConfig  *models.WorkerConfig
	lastVersion string
	subscription *nats.Subscription
	config      natspkg.Config
}

func NewNatsDistributor(natsConfig natspkg.Config, workerMgr *worker.Manager) (*NatsDistributor, error) {
	natsClient := natspkg.NewClient(natsConfig)
	
	err := natsClient.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &NatsDistributor{
		natsClient: natsClient,
		workerMgr:  workerMgr,
		ctx:        ctx,
		cancel:     cancel,
		config:     natsConfig,
	}, nil
}

func (nd *NatsDistributor) Start(ctx context.Context) error {
	logger.Log.Info("Starting NATS pub/sub distribution strategy")

	// Subscribe to NATS config changes
	subject := nd.config.Subject
	if subject == "" {
		subject = "config.worker.update"
	}

	queueGroup := nd.config.QueueGroup
	if queueGroup == "" {
		queueGroup = "config-workers"
	}

	var err error
	nd.subscription, err = nd.natsClient.QueueSubscribe(subject, queueGroup, nd.handleNatsMessage)
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS subject %s: %w", subject, err)
	}

	logger.Log.Infof("NATS subscriber started on subject: %s, queue: %s", subject, queueGroup)

	// Wait for context cancellation
	<-ctx.Done()
	return ctx.Err()
}

func (nd *NatsDistributor) handleNatsMessage(msg *nats.Msg) {
	logger.Log.Debugf("Received NATS message: %s", string(msg.Data))

	var configMsg struct {
		Version string              `json:"version"`
		Config  models.WorkerConfig `json:"config"`
	}

	if err := json.Unmarshal(msg.Data, &configMsg); err != nil {
		logger.Log.Errorf("Failed to unmarshal NATS config message: %v", err)
		return
	}

	nd.mu.Lock()
	defer nd.mu.Unlock()

	// Check if this is a new version
	if configMsg.Version != nd.lastVersion {
		nd.lastConfig = &configMsg.Config
		nd.lastVersion = configMsg.Version

		logger.Log.Infof("Received new config from NATS: version %s", configMsg.Version)

		// Forward to worker
		if err := nd.workerMgr.ForwardConfig(configMsg.Config); err != nil {
			logger.Log.Errorf("Failed to forward NATS config to worker: %v", err)
		} else {
			logger.Log.Info("Successfully forwarded NATS config to worker")
		}
	}
}

func (nd *NatsDistributor) Stop() error {
	logger.Log.Info("Stopping NATS distributor")
	nd.cancel()
	
	if nd.subscription != nil {
		if err := nd.subscription.Unsubscribe(); err != nil {
			logger.Log.Warnf("Failed to unsubscribe from NATS: %v", err)
		}
	}
	
	if nd.natsClient != nil {
		nd.natsClient.Close()
	}
	return nil
}

func (nd *NatsDistributor) GetType() DistributionStrategy {
	return StrategyNats
}

func (nd *NatsDistributor) GetLastConfig() *models.WorkerConfig {
	nd.mu.RLock()
	defer nd.mu.RUnlock()
	return nd.lastConfig
}

func (nd *NatsDistributor) GetLastVersion() string {
	nd.mu.RLock()
	defer nd.mu.RUnlock()
	return nd.lastVersion
}

// DistributionManager manages the selected distribution strategy
type DistributionManager struct {
	strategy    DistributionStrategy
	distributor ConfigDistributor
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewDistributionManager creates a new distribution manager with the specified strategy
func NewDistributionManager(
	strategy DistributionStrategy,
	controllerURL, username, password string,
	workerMgr *worker.Manager,
	cacheFile string,
	redisConfig redis.Config,
	natsConfig natspkg.Config,
) (*DistributionManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	var distributor ConfigDistributor
	var err error

	switch strategy {
	case StrategyPoller:
		distributor = NewPollerDistributor(controllerURL, username, password, workerMgr, cacheFile)
	case StrategyRedis:
		distributor, err = NewRedisDistributor(redisConfig, workerMgr)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create Redis distributor: %w", err)
		}
	case StrategyNats:
		distributor, err = NewNatsDistributor(natsConfig, workerMgr)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create NATS distributor: %w", err)
		}
	default:
		cancel()
		return nil, fmt.Errorf("unsupported distribution strategy: %s", strategy)
	}

	return &DistributionManager{
		strategy:    strategy,
		distributor: distributor,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Start begins the distribution process using the selected strategy
func (dm *DistributionManager) Start() error {
	logger.Log.Infof("Starting distribution manager with strategy: %s", dm.strategy)
	return dm.distributor.Start(dm.ctx)
}

// Stop stops the distribution manager
func (dm *DistributionManager) Stop() error {
	logger.Log.Info("Stopping distribution manager")
	dm.cancel()
	return dm.distributor.Stop()
}

// GetStrategy returns the current distribution strategy
func (dm *DistributionManager) GetStrategy() DistributionStrategy {
	return dm.strategy
}