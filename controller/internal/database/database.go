package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doniyusdinar/config-management/pkg/models"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := conn.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return db, nil
}

// migrate creates the database schema
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS agents (
		id TEXT PRIMARY KEY,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_poll TIMESTAMP,
		metadata TEXT
	);

	CREATE TABLE IF NOT EXISTS configurations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version INTEGER UNIQUE NOT NULL,
		config_data TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS active_config (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		version INTEGER NOT NULL,
		config_data TEXT NOT NULL,
		poll_interval_seconds INTEGER DEFAULT 30,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.conn.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Initialize active_config if it doesn't exist
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM active_config").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check active_config: %w", err)
	}

	if count == 0 {
		defaultConfig := models.WorkerConfig{URL: "https://ip.me"}
		configJSON, _ := json.Marshal(defaultConfig)
		_, err = db.conn.Exec(`
			INSERT INTO active_config (id, version, config_data, poll_interval_seconds)
			VALUES (1, 1, ?, 30)
		`, string(configJSON))
		if err != nil {
			return fmt.Errorf("failed to initialize active_config: %w", err)
		}
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// RegisterAgent registers a new agent
func (db *DB) RegisterAgent(agent *models.Agent) error {
	_, err := db.conn.Exec(`
		INSERT INTO agents (id, registered_at, metadata)
		VALUES (?, ?, ?)
	`, agent.ID, agent.RegisteredAt, agent.Metadata)
	return err
}

// UpdateAgentPoll updates the last poll time for an agent
func (db *DB) UpdateAgentPoll(agentID string) error {
	_, err := db.conn.Exec(`
		UPDATE agents SET last_poll = ? WHERE id = ?
	`, time.Now(), agentID)
	return err
}

// GetActiveConfig retrieves the current active configuration
func (db *DB) GetActiveConfig() (*models.ConfigResponse, error) {
	var version int64
	var configData string
	var pollInterval int

	err := db.conn.QueryRow(`
		SELECT version, config_data, poll_interval_seconds FROM active_config WHERE id = 1
	`).Scan(&version, &configData, &pollInterval)
	if err != nil {
		return nil, err
	}

	var workerConfig models.WorkerConfig
	if err := json.Unmarshal([]byte(configData), &workerConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &models.ConfigResponse{
		Version:          version,
		Data:             workerConfig,
		PollIntervalSecs: pollInterval,
	}, nil
}

// UpdateConfig updates the active configuration
func (db *DB) UpdateConfig(config models.WorkerConfig, pollInterval int) (int64, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Get current version
	var currentVersion int64
	err = db.conn.QueryRow("SELECT version FROM active_config WHERE id = 1").Scan(&currentVersion)
	if err != nil {
		return 0, err
	}

	newVersion := currentVersion + 1

	// Update active config
	_, err = db.conn.Exec(`
		UPDATE active_config 
		SET version = ?, config_data = ?, poll_interval_seconds = ?, updated_at = ?
		WHERE id = 1
	`, newVersion, string(configJSON), pollInterval, time.Now())
	if err != nil {
		return 0, err
	}

	// Store in history
	_, err = db.conn.Exec(`
		INSERT INTO configurations (version, config_data, created_at)
		VALUES (?, ?, ?)
	`, newVersion, string(configJSON), time.Now())
	if err != nil {
		return 0, err
	}

	return newVersion, nil
}

// GetAllAgents retrieves all registered agents
func (db *DB) GetAllAgents() ([]models.Agent, error) {
	rows, err := db.conn.Query(`
		SELECT id, registered_at, last_poll, metadata FROM agents ORDER BY registered_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var agent models.Agent
		var lastPoll sql.NullTime
		err := rows.Scan(&agent.ID, &agent.RegisteredAt, &lastPoll, &agent.Metadata)
		if err != nil {
			return nil, err
		}
		if lastPoll.Valid {
			agent.LastPoll = lastPoll.Time
		}
		agents = append(agents, agent)
	}

	return agents, nil
}
