package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Health check
	router.GET("/health", handler.HealthCheck)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Agent endpoints (agent auth required)
		v1.POST("/register", handler.AgentAuthMiddleware(), handler.RegisterAgent)
		v1.GET("/config", handler.AgentAuthMiddleware(), handler.GetConfig)

		// Admin endpoints (admin auth required)
		v1.POST("/config", handler.AdminAuthMiddleware(), handler.UpdateConfig)
		v1.GET("/agents", handler.AdminAuthMiddleware(), handler.GetAgents)
	}

	return router
}
