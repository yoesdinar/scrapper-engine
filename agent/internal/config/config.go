package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	ControllerURL         string
	ControllerUsername    string
	ControllerPassword    string
	WorkerURL             string
	LogLevel              string
	CacheFile             string
	// Distribution strategy configuration
	DistributionStrategy  string // POLLER, REDIS, NATS, KAFKA (future)
	RedisAddress          string
	RedisPassword         string
	RedisDB               int
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
	viper.SetDefault("DISTRIBUTION_STRATEGY", "POLLER")
	viper.SetDefault("REDIS_ADDRESS", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", "0")

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found; using env vars or defaults
	}

	config := &Config{
		ControllerURL:         getEnv("CONTROLLER_URL", viper.GetString("CONTROLLER_URL")),
		ControllerUsername:    getEnv("CONTROLLER_USERNAME", viper.GetString("CONTROLLER_USERNAME")),
		ControllerPassword:    getEnv("CONTROLLER_PASSWORD", viper.GetString("CONTROLLER_PASSWORD")),
		WorkerURL:             getEnv("WORKER_URL", viper.GetString("WORKER_URL")),
		LogLevel:              getEnv("LOG_LEVEL", viper.GetString("LOG_LEVEL")),
		CacheFile:             getEnv("CACHE_FILE", viper.GetString("CACHE_FILE")),
		DistributionStrategy:  getEnv("DISTRIBUTION_STRATEGY", viper.GetString("DISTRIBUTION_STRATEGY")),
		RedisAddress:          getEnv("REDIS_ADDRESS", viper.GetString("REDIS_ADDRESS")),
		RedisPassword:         getEnv("REDIS_PASSWORD", viper.GetString("REDIS_PASSWORD")),
		RedisDB:               getEnvInt("REDIS_DB", viper.GetInt("REDIS_DB")),
	}

	return config, nil
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
