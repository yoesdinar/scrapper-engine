package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doniyusdinar/config-management/controller/internal/api"
	"github.com/doniyusdinar/config-management/controller/internal/database"
	"github.com/doniyusdinar/config-management/pkg/logger"
)

// @title Configuration Management Controller API
// @version 1.0
// @description Central configuration management service for distributed systems
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.basic BasicAuth

func main() {
	// Initialize logger
	logLevel := getEnv("LOG_LEVEL", "info")
	logger.SetLevel(logLevel)
	logger.Log.Info("Starting Configuration Management Controller")

	// Get configuration from environment
	dbPath := getEnv("DB_PATH", "./controller.db")
	port := getEnv("PORT", "8080")

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		logger.Log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	logger.Log.Info("Database initialized successfully")

	// Initialize API handler
	handler := api.NewHandler(db)

	// Setup router
	router := api.SetupRouter(handler)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Log.Infof("Controller listening on port %s", port)
		logger.Log.Infof("Swagger docs available at http://localhost:%s/swagger/index.html", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Log.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Log.Info("Server exited")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
