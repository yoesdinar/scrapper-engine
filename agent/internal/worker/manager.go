package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
)

type Manager struct {
	workerURL string
	client    *http.Client
}

func NewManager(workerURL string) *Manager {
	return &Manager{
		workerURL: workerURL,
		client:    &http.Client{},
	}
}

// ForwardConfig forwards configuration to the worker
func (m *Manager) ForwardConfig(config models.WorkerConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	url := fmt.Sprintf("%s/config", m.workerURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(configJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to forward config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("worker returned status %d", resp.StatusCode)
	}

	logger.Log.Info("Configuration forwarded to worker successfully")
	return nil
}
