package database

import (
	"os"
	"testing"
	"time"

	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// Create a temporary database file
	dbPath := "test_controller.db"
	db, err := New(dbPath)
	require.NoError(t, err)

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestRegisterAgent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	agent := &models.Agent{
		ID:           "test-agent-123",
		RegisteredAt: time.Now(),
	}
	err := db.RegisterAgent(agent)
	assert.NoError(t, err)
}

func TestGetActiveConfig(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Database initializes with default config
	config, err := db.GetActiveConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "https://ip.me", config.Data.URL)
	assert.Equal(t, int64(1), config.Version)
	assert.Equal(t, 30, config.PollIntervalSecs)

	// Update config
	workerConfig := models.WorkerConfig{URL: "https://example.com"}
	newVersion, err := db.UpdateConfig(workerConfig, 60)
	require.NoError(t, err)

	// Verify updated config
	config, err = db.GetActiveConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "https://example.com", config.Data.URL)
	assert.Equal(t, newVersion, config.Version)
	assert.Equal(t, 60, config.PollIntervalSecs)
}

func TestUpdateConfig(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name         string
		workerConfig models.WorkerConfig
		pollInterval int
		wantVersion  int64
	}{
		{
			name:         "Update to first URL",
			workerConfig: models.WorkerConfig{URL: "https://example.com"},
			pollInterval: 30,
			wantVersion:  2, // Version 1 is default
		},
		{
			name:         "Update to second URL",
			workerConfig: models.WorkerConfig{URL: "https://example.org"},
			pollInterval: 60,
			wantVersion:  3,
		},
		{
			name:         "Update to third URL",
			workerConfig: models.WorkerConfig{URL: "https://test.com"},
			pollInterval: 15,
			wantVersion:  4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := db.UpdateConfig(tt.workerConfig, tt.pollInterval)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantVersion, version)

			config, err := db.GetActiveConfig()
			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Equal(t, tt.workerConfig.URL, config.Data.URL)
			assert.Equal(t, tt.wantVersion, config.Version)
			assert.Equal(t, tt.pollInterval, config.PollIntervalSecs)
		})
	}
}

func TestGetAllAgents(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Initially should have no agents (or just default)
	agents, err := db.GetAllAgents()
	assert.NoError(t, err)
	initialCount := len(agents)

	// Register some agents
	agentIDs := []string{"agent-1", "agent-2", "agent-3"}
	for _, id := range agentIDs {
		agent := &models.Agent{
			ID:           id,
			RegisteredAt: time.Now(),
		}
		err := db.RegisterAgent(agent)
		require.NoError(t, err)
	}

	// Now should return all agents
	agents, err = db.GetAllAgents()
	assert.NoError(t, err)
	assert.Len(t, agents, initialCount+3)
}

func TestConfigVersionIncrement(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Get initial version (should be 1 from default)
	initialConfig, err := db.GetActiveConfig()
	require.NoError(t, err)
	assert.Equal(t, int64(1), initialConfig.Version)

	// Update config multiple times and check version increment
	for i := 2; i <= 5; i++ {
		workerConfig := models.WorkerConfig{URL: "https://example.com"}
		version, err := db.UpdateConfig(workerConfig, 30)
		require.NoError(t, err)
		assert.Equal(t, int64(i), version)

		config, err := db.GetActiveConfig()
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, int64(i), config.Version)
	}
}
