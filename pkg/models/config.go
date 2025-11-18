package models
package models

import "time"

// WorkerConfig represents the configuration that workers execute
type WorkerConfig struct {
	URL string `json:"url" validate:"required,url"`
}

// Config represents a configuration version stored in the database
type Config struct {
	ID        int64         `json:"id"`
	Version   int64         `json:"version"`
	Data      WorkerConfig  `json:"data"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// ConfigResponse represents the response when fetching config
type ConfigResponse struct {
	Version          int64        `json:"version"`
	Data             WorkerConfig `json:"data"`
	PollIntervalSecs int          `json:"poll_interval_seconds,omitempty"`
}
