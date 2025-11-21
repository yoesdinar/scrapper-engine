package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/doniyusdinar/config-management/controller/internal/database"
	"github.com/doniyusdinar/config-management/pkg/auth"
	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/doniyusdinar/config-management/pkg/redis"
	natspkg "github.com/doniyusdinar/config-management/pkg/nats"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	db            *database.DB
	redisClient   *redis.Client
	natsClient    *natspkg.Client
	agentUsername string
	agentPassword string
	adminUsername string
	adminPassword string
	pollInterval  int
}

func NewHandler(db *database.DB, redisClient *redis.Client, natsClient *natspkg.Client) *Handler {
	return &Handler{
		db:            db,
		redisClient:   redisClient,
		natsClient:    natsClient,
		agentUsername: getEnv("AGENT_USERNAME", "agent"),
		agentPassword: getEnv("AGENT_PASSWORD", "secret123"),
		adminUsername: getEnv("ADMIN_USERNAME", "admin"),
		adminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		pollInterval:  getEnvInt("DEFAULT_POLL_INTERVAL", 30),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// AgentAuthMiddleware validates agent credentials
func (h *Handler) AgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !auth.ValidateBasicAuth(authHeader, h.agentUsername, h.agentPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// AdminAuthMiddleware validates admin credentials
func (h *Handler) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !auth.ValidateBasicAuth(authHeader, h.adminUsername, h.adminPassword) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RegisterAgent godoc
// @Summary Register a new agent
// @Description Register a new agent with the controller
// @Tags agents
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Agent registration request"
// @Success 200 {object} models.RegisterResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/register [post]
// @Security BasicAuth
func (h *Handler) RegisterAgent(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Log.Errorf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	agentID := uuid.New().String()
	agent := &models.Agent{
		ID:       agentID,
		Metadata: req.Metadata,
	}

	if err := h.db.RegisterAgent(agent); err != nil {
		logger.Log.Errorf("Failed to register agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register agent"})
		return
	}

	logger.Log.Infof("Agent registered: %s", agentID)

	response := models.RegisterResponse{
		AgentID:          agentID,
		PollURL:          "/api/v1/config",
		PollIntervalSecs: h.pollInterval,
	}

	c.JSON(http.StatusOK, response)
}

// GetConfig godoc
// @Summary Get current configuration
// @Description Get the current active configuration for agents
// @Tags config
// @Produce json
// @Success 200 {object} models.ConfigResponse
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/config [get]
// @Security BasicAuth
func (h *Handler) GetConfig(c *gin.Context) {
	config, err := h.db.GetActiveConfig()
	if err != nil {
		logger.Log.Errorf("Failed to get config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get config"})
		return
	}

	c.Header("ETag", strconv.FormatInt(config.Version, 10))
	c.JSON(http.StatusOK, config)
}

// UpdateConfig godoc
// @Summary Update configuration
// @Description Update the global configuration (admin only)
// @Tags config
// @Accept json
// @Produce json
// @Param config body models.WorkerConfig true "New configuration"
// @Param poll_interval query int false "Poll interval in seconds" default(30)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/config [post]
// @Security BasicAuth
func (h *Handler) UpdateConfig(c *gin.Context) {
	var config models.WorkerConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		logger.Log.Errorf("Invalid config: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config"})
		return
	}

	pollInterval := h.pollInterval
	if pi := c.Query("poll_interval"); pi != "" {
		if val, err := strconv.Atoi(pi); err == nil && val > 0 {
			pollInterval = val
		}
	}

	version, err := h.db.UpdateConfig(config, pollInterval)
	if err != nil {
		logger.Log.Errorf("Failed to update config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	logger.Log.Infof("Configuration updated to version %d", version)

	// Publish to Redis if available (non-blocking)
	if h.redisClient != nil && h.redisClient.IsConnected() {
		versionStr := strconv.Itoa(int(version))
		if err := h.redisClient.PublishConfig(config, versionStr); err != nil {
			logger.Log.Warnf("Failed to publish config to Redis (continuing with polling): %v", err)
		} else {
			// Store backup in Redis
			h.redisClient.StoreConfigInRedis(config, versionStr)
			logger.Log.Info("Configuration published to Redis successfully")
		}
	}

	// Publish to NATS if available (non-blocking)
	if h.natsClient != nil && h.natsClient.IsConnected() {
		versionStr := strconv.Itoa(int(version))
		configMessage := struct {
			Version string              `json:"version"`
			Config  models.WorkerConfig `json:"config"`
		}{
			Version: versionStr,
			Config:  config,
		}

		messageData, err := json.Marshal(configMessage)
		if err != nil {
			logger.Log.Warnf("Failed to marshal config for NATS: %v", err)
		} else {
			subject := "config.worker.update" // Default subject

			if err := h.natsClient.Publish(subject, messageData); err != nil {
				logger.Log.Warnf("Failed to publish config to NATS (continuing with polling): %v", err)
			} else {
				logger.Log.Infof("Configuration published to NATS successfully on subject: %s", subject)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"version": version,
	})
}

// GetAgents godoc
// @Summary Get all registered agents
// @Description Get a list of all registered agents (admin only)
// @Tags agents
// @Produce json
// @Success 200 {array} models.Agent
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/agents [get]
// @Security BasicAuth
func (h *Handler) GetAgents(c *gin.Context) {
	agents, err := h.db.GetAllAgents()
	if err != nil {
		logger.Log.Errorf("Failed to get agents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agents"})
		return
	}

	c.JSON(http.StatusOK, agents)
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the service is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
