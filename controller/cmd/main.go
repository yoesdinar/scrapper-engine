package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/doniyusdinar/config-management/controller/internal/api"
	"github.com/doniyusdinar/config-management/controller/internal/database"
	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/pkg/redis"
	natspkg "github.com/doniyusdinar/config-management/pkg/nats"

	_ "github.com/doniyusdinar/config-management/controller/docs"
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
	logLevel := getEnv("LOG_LEVEL", "info")
	logger.SetLevel(logLevel)
	logger.Log.Info("Starting Configuration Management Controller")

	dbPath := getEnv("DB_PATH", "./controller.db")
	port := getEnv("PORT", "8080")

	db, err := database.New(dbPath)
	if err != nil {
		logger.Log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	logger.Log.Info("Database initialized successfully")

	// Initialize Redis client (optional based on distribution strategy)
	var redisClient *redis.Client
	var natsClient *natspkg.Client
	distributionStrategy := getEnv("DISTRIBUTION_STRATEGY", "POLLER")
	
	if distributionStrategy == "REDIS" {
		redisConfig := redis.Config{
			Address:  getEnv("REDIS_ADDRESS", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
			Enabled:  true,
		}

		redisClient, err = redis.NewClient(redisConfig)
		if err != nil {
			logger.Log.Warnf("Failed to connect to Redis, Redis distribution disabled: %v", err)
			redisClient = nil
		} else {
			defer redisClient.Close()
			logger.Log.Info("Redis client initialized successfully for distribution")
		}
	} else if distributionStrategy == "NATS" {
		natsConfig := natspkg.Config{
			URLs:            strings.Split(getEnv("NATS_URL", "nats://localhost:4222"), ","),
			Username:        getEnv("NATS_USERNAME", ""),
			Password:        getEnv("NATS_PASSWORD", ""),
			Token:           getEnv("NATS_TOKEN", ""),
			TLSEnabled:      getEnvBool("NATS_TLS_ENABLED", false),
			MaxReconnect:    10,
			ReconnectWait:   2 * time.Second,
			ConnectionName:  "controller-publisher",
			Subject:         getEnv("NATS_SUBJECT", "config.worker.update"),
			QueueGroup:      getEnv("NATS_QUEUE_GROUP", "config-workers"),
			Enabled:         true,
		}

		natsClient = natspkg.NewClient(natsConfig)
		err = natsClient.Connect()
		if err != nil {
			logger.Log.Warnf("Failed to connect to NATS, NATS distribution disabled: %v", err)
			natsClient = nil
		} else {
			defer natsClient.Close()
			logger.Log.Info("NATS client initialized successfully for distribution")
		}
	} else {
		logger.Log.Infof("Distribution strategy: %s (Redis/NATS not needed)", distributionStrategy)
	}

	handler := api.NewHandler(db, redisClient, natsClient)
	router := api.SetupRouter(handler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		logger.Log.Infof("Controller listening on port %s", port)
		logger.Log.Infof("Swagger docs available at http://localhost:%s/swagger/index.html", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")

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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
