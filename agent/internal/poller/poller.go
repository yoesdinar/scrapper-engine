package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/doniyusdinar/config-management/agent/internal/backoff"
	"github.com/doniyusdinar/config-management/agent/internal/worker"
	"github.com/doniyusdinar/config-management/pkg/auth"
	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
)

type Poller struct {
	controllerURL string
	authHeader    string
	client        *http.Client
	workerMgr     *worker.Manager
	backoff       *backoff.Backoff
	cacheFile     string

	currentVersion   int64
	pollInterval     time.Duration
	updateIntervalCh chan time.Duration
}

func NewPoller(controllerURL, username, password string, workerMgr *worker.Manager, cacheFile string) *Poller {
	return &Poller{
		controllerURL:    controllerURL,
		authHeader:       auth.CreateBasicAuthHeader(username, password),
		client:           &http.Client{Timeout: 10 * time.Second},
		workerMgr:        workerMgr,
		backoff:          backoff.New(1*time.Second, 5*time.Minute, 2.0),
		cacheFile:        cacheFile,
		pollInterval:     30 * time.Second, // Default, will be updated by controller
		updateIntervalCh: make(chan time.Duration, 1),
	}
}

// Start starts the polling loop
func (p *Poller) Start(ctx context.Context) error {
	if err := p.loadCache(); err != nil {
		logger.Log.Warnf("Failed to load cache: %v", err)
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Polling stopped")
			return nil

		case newInterval := <-p.updateIntervalCh:
			logger.Log.Infof("Updating poll interval to %v", newInterval)
			p.pollInterval = newInterval
			ticker.Reset(newInterval)

		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				logger.Log.Errorf("Poll failed: %v", err)

				backoffDuration := p.backoff.Next()
				logger.Log.Infof("Retrying in %v", backoffDuration)

				select {
				case <-time.After(backoffDuration):
				case <-ctx.Done():
					return nil
				}
			} else {
				p.backoff.Reset()
			}
		}
	}
}

// poll fetches configuration from controller
func (p *Poller) poll(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/config", p.controllerURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", p.authHeader)
	if p.currentVersion > 0 {
		req.Header.Set("If-None-Match", strconv.FormatInt(p.currentVersion, 10))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		logger.Log.Debug("Configuration unchanged")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("controller returned status %d: %s", resp.StatusCode, string(body))
	}

	var configResp models.ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return fmt.Errorf("failed to decode config: %w", err)
	}

	if configResp.Version != p.currentVersion {
		logger.Log.Infof("Configuration changed: version %d -> %d", p.currentVersion, configResp.Version)
		p.currentVersion = configResp.Version

		if err := p.workerMgr.ForwardConfig(configResp.Data); err != nil {
			logger.Log.Errorf("Failed to forward config to worker: %v", err)
			return err
		}

		if err := p.saveCache(configResp); err != nil {
			logger.Log.Warnf("Failed to save cache: %v", err)
		}
	}

	if configResp.PollIntervalSecs > 0 {
		newInterval := time.Duration(configResp.PollIntervalSecs) * time.Second
		if newInterval != p.pollInterval {
			select {
			case p.updateIntervalCh <- newInterval:
			default:
			}
		}
	}

	return nil
}

// loadCache loads configuration from cache file
func (p *Poller) loadCache() error {
	data, err := os.ReadFile(p.cacheFile)
	if err != nil {
		return err
	}

	var configResp models.ConfigResponse
	if err := json.Unmarshal(data, &configResp); err != nil {
		return err
	}

	p.currentVersion = configResp.Version

	if err := p.workerMgr.ForwardConfig(configResp.Data); err != nil {
		return err
	}

	logger.Log.Infof("Loaded cached config version %d", configResp.Version)
	return nil
}

// saveCache saves configuration to cache file
func (p *Poller) saveCache(config models.ConfigResponse) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(p.cacheFile, data, 0644)
}
