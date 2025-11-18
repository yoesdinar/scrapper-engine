package config

import (
"testing"

"github.com/doniyusdinar/config-management/pkg/models"
"github.com/stretchr/testify/assert"
)

func TestGetConfigEmpty(t *testing.T) {
	mgr := NewManager()
	
	_, hasConfig := mgr.GetConfig()
	assert.False(t, hasConfig)
	assert.False(t, mgr.HasConfig())
}

func TestUpdateAndGetConfig(t *testing.T) {
	mgr := NewManager()
	
	config := models.WorkerConfig{URL: "https://example.com"}
	mgr.UpdateConfig(config)
	
	retrieved, hasConfig := mgr.GetConfig()
	assert.True(t, hasConfig)
	assert.True(t, mgr.HasConfig())
	assert.Equal(t, "https://example.com", retrieved.URL)
}

func TestUpdateConfigMultipleTimes(t *testing.T) {
	mgr := NewManager()
	
	urls := []string{
		"https://example.com",
		"https://test.com",
		"https://final.com",
	}
	
	for _, url := range urls {
		config := models.WorkerConfig{URL: url}
		mgr.UpdateConfig(config)
		
		retrieved, hasConfig := mgr.GetConfig()
		assert.True(t, hasConfig)
		assert.Equal(t, url, retrieved.URL)
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr := NewManager()
	
	// Start multiple goroutines writing
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			config := models.WorkerConfig{URL: "https://example.com"}
			mgr.UpdateConfig(config)
			done <- true
		}(i)
	}
	
	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify we can still read
	retrieved, hasConfig := mgr.GetConfig()
	assert.True(t, hasConfig)
	assert.Equal(t, "https://example.com", retrieved.URL)
}
