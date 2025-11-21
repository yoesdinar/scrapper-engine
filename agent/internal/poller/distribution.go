package poller

import (
	"context"
	"fmt"
	"sync"

	"github.com/doniyusdinar/config-management/agent/internal/worker"
	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/doniyusdinar/config-management/pkg/redis"
)

// DistributionStrategy represents the config distribution strategy
type DistributionStrategy string

const (
	StrategyPoller DistributionStrategy = "POLLER"
	StrategyRedis  DistributionStrategy = "REDIS"
	// Future strategies:
	// StrategyNats   DistributionStrategy = "NATS"
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