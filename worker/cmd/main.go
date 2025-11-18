package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/doniyusdinar/config-management/pkg/logger"
	"github.com/doniyusdinar/config-management/worker/internal/api"
	"github.com/doniyusdinar/config-management/worker/internal/config"
	"github.com/doniyusdinar/config-management/worker/internal/proxy"

	_ "github.com/doniyusdinar/config-management/worker/docs"
)

// @title Configuration Management Worker API
// @version 1.0
// @description Worker service that executes tasks based on received configuration
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8082
// @BasePath /
// @schemes http https

func main() {
	logLevel := getEnv("LOG_LEVEL", "info")
	logger.SetLevel(logLevel)
	logger.Log.Info("Starting Configuration Management Worker")

	port := getEnv("PORT", "8082")

	configMgr := config.NewManager()
	proxyClient := proxy.NewProxy()

	handler := api.NewHandler(configMgr, proxyClient)
	router := api.SetupRouter(handler)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		logger.Log.Infof("Worker listening on port %s", port)
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
