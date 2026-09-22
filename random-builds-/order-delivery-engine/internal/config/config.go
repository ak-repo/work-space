package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Environment   string              `yaml:"environment"`
	Server        ServerConfig        `yaml:"server"`
	Database      DatabaseConfig      `yaml:"database"`
	Assignment    AssignmentConfig    `yaml:"assignment"`
	SSE           SSEConfig           `yaml:"sse"`
	Defaults      DefaultsConfig      `yaml:"defaults"`
	Observability ObservabilityConfig `yaml:"observability"`
}

type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            string        `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            string        `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	DBName          string        `yaml:"dbname"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type AssignmentConfig struct {
	MaxSearchRadiusKm float64 `yaml:"max_search_radius_km"`
	DistanceWeight    float64 `yaml:"distance_weight"`
	CapacityWeight    float64 `yaml:"capacity_weight"`
	DistanceSmoothing float64 `yaml:"distance_smoothing"`
}

type SSEConfig struct {
	ClientChannelBuffer int `yaml:"client_channel_buffer"`
}

type DefaultsConfig struct {
	DriverCapacity int `yaml:"driver_capacity"`
	OrderPriority  int `yaml:"order_priority"`
}

type ObservabilityConfig struct {
	ServiceName            string        `yaml:"service_name"`
	ServiceVersion         string        `yaml:"service_version"`
	LogLevel               string        `yaml:"log_level"`
	LogFormat              string        `yaml:"log_format"`
	GinStyleAccess         bool          `yaml:"gin_style_access"`
	AddSource              bool          `yaml:"add_source"`
	RequestIDHeader        string        `yaml:"request_id_header"`
	TrustIncomingRequestID bool          `yaml:"trust_incoming_request_id"`
	LogQueryString         bool          `yaml:"log_query_string"`
	LogUserAgent           bool          `yaml:"log_user_agent"`
	SlowRequestThreshold   time.Duration `yaml:"slow_request_threshold"`
	SlowQueryThreshold     time.Duration `yaml:"slow_query_threshold"`
	LogAllQueries          bool          `yaml:"log_all_queries"`
	LogQueryArgs           bool          `yaml:"log_query_args"`
	MaxSQLLength           int           `yaml:"max_sql_length"`
	MaxPathLength          int           `yaml:"max_path_length"`
	AuditEnabled           bool          `yaml:"audit_enabled"`
	LogSSEConnect          bool          `yaml:"log_sse_connect"`
	LogSSEDisconnect       bool          `yaml:"log_sse_disconnect"`
	WarnOnDroppedSSEEvent  bool          `yaml:"warn_on_dropped_sse_event"`
}

// Load reads the config file and applies environment variable overrides
func Load() (*Config, error) {
	// Determine config file path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "internal/config/config.yaml"
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	return &cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("APP_ENV"); v != "" {
		c.Environment = v
	}

	// Server overrides
	if v := os.Getenv("HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("PORT"); v != "" {
		c.Server.Port = v
	}

	// Database overrides
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		c.Database.Port = v
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}

	// Observability overrides
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		c.Observability.LogLevel = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		c.Observability.LogFormat = v
	}
	if v := os.Getenv("HTTP_SLOW_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Observability.SlowRequestThreshold = d
		}
	}
	if v := os.Getenv("DB_SLOW_THRESHOLD"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			c.Observability.SlowQueryThreshold = d
		}
	}
}
