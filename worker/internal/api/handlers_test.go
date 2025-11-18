package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/doniyusdinar/config-management/worker/internal/config"
	"github.com/doniyusdinar/config-management/worker/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestHandler() (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	configMgr := config.NewManager()
	proxyObj := proxy.NewProxy()
	handler := NewHandler(configMgr, proxyObj)
	router := gin.New()

	return handler, router
}

func TestHealthCheck(t *testing.T) {
	handler, router := setupTestHandler()
	router.GET("/health", handler.HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
	assert.Equal(t, false, response["has_config"])
}

func TestUpdateConfig(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/config", handler.UpdateConfig)

	workerConfig := models.WorkerConfig{URL: "https://example.com"}
	body, _ := json.Marshal(workerConfig)

	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Configuration updated", response["message"])

	// Verify config was stored
	storedConfig, hasConfig := handler.configMgr.GetConfig()
	assert.True(t, hasConfig)
	assert.Equal(t, "https://example.com", storedConfig.URL)
}

func TestUpdateConfigInvalidJSON(t *testing.T) {
	handler, router := setupTestHandler()
	router.POST("/config", handler.UpdateConfig)

	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHitNoConfig(t *testing.T) {
	handler, router := setupTestHandler()
	router.GET("/hit", handler.Hit)

	req := httptest.NewRequest(http.MethodGet, "/hit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "No configuration available", response["error"])
}

func TestHitWithConfig(t *testing.T) {
	handler, router := setupTestHandler()
	router.GET("/hit", handler.Hit)

	// Create a test server to act as the configured URL
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer targetServer.Close()

	// Set config
	workerConfig := models.WorkerConfig{URL: targetServer.URL}
	handler.configMgr.UpdateConfig(workerConfig)

	req := httptest.NewRequest(http.MethodGet, "/hit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

func TestHitWithInvalidURL(t *testing.T) {
	handler, router := setupTestHandler()
	router.GET("/hit", handler.Hit)

	// Set config with invalid URL
	workerConfig := models.WorkerConfig{URL: "http://invalid-url-does-not-exist:9999"}
	handler.configMgr.UpdateConfig(workerConfig)

	req := httptest.NewRequest(http.MethodGet, "/hit", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
