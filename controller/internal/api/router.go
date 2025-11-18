package api

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func SetupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.GET("/health", handler.HealthCheck)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/api/v1")
	{
		v1.POST("/register", handler.AgentAuthMiddleware(), handler.RegisterAgent)
		v1.GET("/config", handler.AgentAuthMiddleware(), handler.GetConfig)
		v1.POST("/config", handler.AdminAuthMiddleware(), handler.UpdateConfig)
		v1.GET("/agents", handler.AdminAuthMiddleware(), handler.GetAgents)
	}

	return router
}
