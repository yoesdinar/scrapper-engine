package worker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForwardConfig(t *testing.T) {
	// Create a test server to act as worker
	called := false
	var receivedConfig models.WorkerConfig

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "/config", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		// Decode the config
		err := json.NewDecoder(r.Body).Decode(&receivedConfig)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewManager(server.URL)
	config := models.WorkerConfig{URL: "https://example.com"}

	err := manager.ForwardConfig(config)
	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "https://example.com", receivedConfig.URL)
}

func TestForwardConfigError(t *testing.T) {
	// Use invalid URL to cause error
	manager := NewManager("http://invalid-url-that-does-not-exist:9999")
	config := models.WorkerConfig{URL: "https://example.com"}

	err := manager.ForwardConfig(config)
	assert.Error(t, err)
}

func TestForwardConfigBadStatus(t *testing.T) {
	// Create a test server that returns error status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	manager := NewManager(server.URL)
	config := models.WorkerConfig{URL: "https://example.com"}

	err := manager.ForwardConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status")
}
