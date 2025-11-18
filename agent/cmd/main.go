package main

import (
	"bytes"
	"context"
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
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Log.Fatalf("Failed to load config: %v", err)
	}

	logger.SetLevel(cfg.LogLevel)
	logger.Log.Info("Starting Configuration Management Agent")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register with controller
	agentID, pollURL, pollInterval, err := registerWithController(cfg)
	if err != nil {
		logger.Log.Fatalf("Failed to register with controller: %v", err)
	}

	logger.Log.Infof("Registered with controller - Agent ID: %s", agentID)
	logger.Log.Infof("Poll URL: %s, Interval: %d seconds", pollURL, pollInterval)

	// Initialize worker manager
	workerMgr := worker.NewManager(cfg.WorkerURL)

	// Initialize poller
	p := poller.NewPoller(cfg.ControllerURL, cfg.ControllerUsername, cfg.ControllerPassword, workerMgr, cfg.CacheFile)

	// Start polling in goroutine
	go func() {
		if err := p.Start(ctx); err != nil {
			logger.Log.Errorf("Poller error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down agent...")
	cancel()
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
