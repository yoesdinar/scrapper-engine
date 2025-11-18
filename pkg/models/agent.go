package models

import "time"

// Agent represents a registered agent in the system
type Agent struct {
	ID           string    `json:"id"`
	RegisteredAt time.Time `json:"registered_at"`
	LastPoll     time.Time `json:"last_poll,omitempty"`
	Metadata     string    `json:"metadata,omitempty"`
}

// RegisterRequest represents the agent registration request
type RegisterRequest struct {
	Hostname string `json:"hostname,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

// RegisterResponse represents the response to agent registration
type RegisterResponse struct {
	AgentID          string `json:"agent_id"`
	PollURL          string `json:"poll_url"`
	PollIntervalSecs int    `json:"poll_interval_seconds"`
}
