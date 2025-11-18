package config

import (
	"sync"

	"github.com/doniyusdinar/config-management/pkg/models"
)

type Manager struct {
	mu        sync.RWMutex
	config    models.WorkerConfig
	hasConfig bool
}

func NewManager() *Manager {
	return &Manager{}
}

// GetConfig returns the current configuration
func (m *Manager) GetConfig() (models.WorkerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config, m.hasConfig
}

// UpdateConfig updates the configuration
func (m *Manager) UpdateConfig(config models.WorkerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
	m.hasConfig = true
}

// HasConfig returns whether configuration has been set
func (m *Manager) HasConfig() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hasConfig
}
