package api

import (
	"net/http"

	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/models"
	"github.com/doniyusdinar/config-management/worker/internal/config"
	"github.com/doniyusdinar/config-management/worker/internal/proxy"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	configMgr *config.Manager
	proxy     *proxy.Proxy
}

func NewHandler(configMgr *config.Manager, proxy *proxy.Proxy) *Handler {
	return &Handler{
		configMgr: configMgr,
		proxy:     proxy,
	}
}

// UpdateConfig godoc
// @Summary Update worker configuration
// @Description Receive configuration update from agent
// @Tags config
// @Accept json
// @Produce json
// @Param config body models.WorkerConfig true "Worker configuration"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /config [post]
func (h *Handler) UpdateConfig(c *gin.Context) {
	var config models.WorkerConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		logger.Log.Errorf("Invalid config: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid config"})
		return
	}

	h.configMgr.UpdateConfig(config)
	logger.Log.Infof("New configuration received: %+v", config)

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated"})
}

// Hit godoc
// @Summary Execute configured task
// @Description Execute HTTP GET request to configured URL and return response
// @Tags task
// @Produce json
// @Produce text/plain
// @Success 200 {string} string "Response from configured URL"
// @Failure 500 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /hit [get]
func (h *Handler) Hit(c *gin.Context) {
	config, hasConfig := h.configMgr.GetConfig()
	if !hasConfig {
		logger.Log.Warn("No configuration available")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No configuration available"})
		return
	}

	logger.Log.Infof("Executing request to: %s", config.URL)

	body, statusCode, err := h.proxy.ExecuteRequest(config.URL)
	if err != nil {
		logger.Log.Errorf("Request failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the response with original status code and content type
	c.Data(statusCode, "text/plain", body)
}

// HealthCheck godoc
// @Summary Health check
// @Description Check if the worker service is running
// @Tags health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	hasConfig := h.configMgr.HasConfig()
	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"has_config": hasConfig,
	})
}
