package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/doniyusdinar/config-management/agent/internal/config"
	"github.com/doniyusdinar/config-management/agent/internal/poller"
	"github.com/doniyusdinar/config-management/agent/internal/worker"
	"github.com/doniyusdinar/config-management/pkg/auth"
	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/doniyusdinar/config-management/pkg/redis"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Log.Fatalf("Failed to load config: %v", err)
	}

	logger.SetLevel(cfg.LogLevel)
	logger.Log.Info("Starting Configuration Management Agent")

	agentID, pollURL, pollInterval, err := registerWithController(cfg)
	if err != nil {
		logger.Log.Fatalf("Failed to register with controller: %v", err)
	}

	logger.Log.Infof("Registered with controller - Agent ID: %s", agentID)
	logger.Log.Infof("Poll URL: %s, Interval: %d seconds", pollURL, pollInterval)

	workerMgr := worker.NewManager(cfg.WorkerURL)

	// Create Redis config for strategy
	redisConfig := redis.Config{
		Address:  cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		Enabled:  true, // Always enabled for Redis strategy
	}

	// Determine distribution strategy
	strategy := poller.DistributionStrategy(cfg.DistributionStrategy)
	
	// Create distribution manager with the specified strategy
	distributionMgr, err := poller.NewDistributionManager(
		strategy,
		cfg.ControllerURL,
		cfg.ControllerUsername,
		cfg.ControllerPassword,
		workerMgr,
		cfg.CacheFile,
		redisConfig,
	)
	if err != nil {
		logger.Log.Fatalf("Failed to create distribution manager: %v", err)
	}

	go func() {
		if err := distributionMgr.Start(); err != nil {
			logger.Log.Errorf("Distribution manager error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down agent...")
	distributionMgr.Stop()
	logger.Log.Info("Agent exited")
}

func registerWithController(cfg *config.Config) (string, string, int, error) {
	url := fmt.Sprintf("%s/api/v1/register", cfg.ControllerURL)

	req := models.RegisterRequest{
		Hostname: getHostname(),
		Metadata: fmt.Sprintf("worker_url=%s", cfg.WorkerURL),
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", auth.CreateBasicAuthHeader(cfg.ControllerUsername, cfg.ControllerPassword))

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("controller returned status %d", resp.StatusCode)
	}

	var registerResp models.RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return "", "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return registerResp.AgentID, registerResp.PollURL, registerResp.PollIntervalSecs, nil
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
