package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/doniyusdinar/config-management/controller/internal/database"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*Handler, *gin.Engine, func()) {
	gin.SetMode(gin.TestMode)

	// Create a temporary database
	dbPath := "test_api.db"
	db, err := database.New(dbPath)
	require.NoError(t, err)

	handler := NewHandler(db)
	router := gin.New()

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return handler, router, cleanup
}

func TestHealthCheck(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.GET("/health", handler.HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

func TestRegisterAgent(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/register", handler.AgentAuthMiddleware(), handler.RegisterAgent)

	reqBody := models.RegisterRequest{
		Hostname: "test-agent",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("agent", "secret123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.RegisterResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.AgentID)
	assert.Equal(t, "/api/v1/config", response.PollURL)
	assert.Equal(t, 30, response.PollIntervalSecs)
}

func TestRegisterAgentUnauthorized(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/register", handler.AgentAuthMiddleware(), handler.RegisterAgent)

	reqBody := models.RegisterRequest{
		Hostname: "test-agent",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// No auth header
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetConfig(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.GET("/config", handler.AgentAuthMiddleware(), handler.GetConfig)

	// Create initial config
	workerConfig := models.WorkerConfig{URL: "https://example.com"}
	_, _ = handler.db.UpdateConfig(workerConfig, 30)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	req.SetBasicAuth("agent", "secret123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.ConfigResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", response.Data.URL)
	assert.Greater(t, response.Version, int64(0))
	assert.Equal(t, 30, response.PollIntervalSecs)
	assert.NotEmpty(t, w.Header().Get("ETag"))
}

func TestUpdateConfig(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/config", handler.AdminAuthMiddleware(), handler.UpdateConfig)

	reqBody := models.WorkerConfig{URL: "https://newurl.com"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("admin", "admin123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify config was updated
	config, err := handler.db.GetActiveConfig()
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "https://newurl.com", config.Data.URL)
}

func TestUpdateConfigUnauthorized(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.POST("/config", handler.AdminAuthMiddleware(), handler.UpdateConfig)

	reqBody := models.WorkerConfig{URL: "https://newurl.com"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/config", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("agent", "agent123") // Wrong credentials
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetAgents(t *testing.T) {
	handler, router, cleanup := setupTestHandler(t)
	defer cleanup()

	router.GET("/agents", handler.AdminAuthMiddleware(), handler.GetAgents)

	// Register some agents
	agent1 := &models.Agent{ID: "agent-1", RegisteredAt: time.Now()}
	agent2 := &models.Agent{ID: "agent-2", RegisteredAt: time.Now()}
	handler.db.RegisterAgent(agent1)
	handler.db.RegisterAgent(agent2)

	req := httptest.NewRequest(http.MethodGet, "/agents", nil)
	req.SetBasicAuth("admin", "admin123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response []models.Agent
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Len(t, response, 2)
}
