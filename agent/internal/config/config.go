package config

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	ControllerURL      string
	ControllerUsername string
	ControllerPassword string
	WorkerURL          string
	LogLevel           string
	CacheFile          string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("CONTROLLER_URL", "http://localhost:8080")
	viper.SetDefault("CONTROLLER_USERNAME", "agent")
	viper.SetDefault("CONTROLLER_PASSWORD", "secret123")
	viper.SetDefault("WORKER_URL", "http://localhost:8082")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("CACHE_FILE", "./agent_config.cache")

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found; using env vars or defaults
	}

	config := &Config{
		ControllerURL:      getEnv("CONTROLLER_URL", viper.GetString("CONTROLLER_URL")),
		ControllerUsername: getEnv("CONTROLLER_USERNAME", viper.GetString("CONTROLLER_USERNAME")),
		ControllerPassword: getEnv("CONTROLLER_PASSWORD", viper.GetString("CONTROLLER_PASSWORD")),
		WorkerURL:          getEnv("WORKER_URL", viper.GetString("WORKER_URL")),
		LogLevel:           getEnv("LOG_LEVEL", viper.GetString("LOG_LEVEL")),
		CacheFile:          getEnv("CACHE_FILE", viper.GetString("CACHE_FILE")),
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
