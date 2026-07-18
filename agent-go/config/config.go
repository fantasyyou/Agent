// Package config loads and validates application configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	MySQL         MySQLConfig         `json:"mysql"`
	Elasticsearch ElasticsearchConfig `json:"elasticsearch"`
	Indices       IndicesConfig       `json:"indices"`
	Auth          AuthConfig          `json:"auth"`
	PythonAgent   PythonAgentConfig   `json:"python_agent"`
	Memory        MemoryConfig        `json:"memory"`
	Redis         RedisConfig         `json:"redis"`
}

type MySQLConfig struct {
	DSN                   string `json:"dsn"`
	MaxOpenConnections    int    `json:"max_open_connections"`
	MaxIdleConnections    int    `json:"max_idle_connections"`
	ConnectionMaxLifetime string `json:"connection_max_lifetime"`
}
type ElasticsearchConfig struct {
	Addresses []string `json:"addresses"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	APIKey    string   `json:"api_key"`
	Timeout   string   `json:"timeout"`
}
type IndicesConfig struct {
	ConversationMemory string `json:"conversation_memory"`
}
type AuthConfig struct {
	SigningSecret string `json:"signing_secret"`
	CookieName    string `json:"cookie_name"`
	TTL           string `json:"ttl"`
	SecureCookie  bool   `json:"secure_cookie"`
}
type PythonAgentConfig struct {
	Address string `json:"address"`
	Timeout string `json:"timeout"`
}
type MemoryConfig struct {
	RecallLimit int `json:"recall_limit"`
}
type RedisConfig struct {
	Address  string `json:"address"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Timeout  string `json:"timeout"`
	StateTTL string `json:"state_ttl"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", path, err)
	}
	if value := os.Getenv("MYSQL_DSN"); value != "" {
		cfg.MySQL.DSN = value
	}
	if value := os.Getenv("AUTH_SIGNING_SECRET"); value != "" {
		cfg.Auth.SigningSecret = value
	}
	if value := os.Getenv("REDIS_ADDRESS"); value != "" {
		cfg.Redis.Address = value
	}
	if value := os.Getenv("REDIS_PASSWORD"); value != "" {
		cfg.Redis.Password = value
	}
	if cfg.MySQL.DSN == "" {
		return Config{}, fmt.Errorf("mysql.dsn is required")
	}
	if len(cfg.Elasticsearch.Addresses) == 0 || cfg.Indices.ConversationMemory == "" {
		return Config{}, fmt.Errorf("elasticsearch.addresses and indices.conversation_memory are required")
	}
	if len(cfg.Auth.SigningSecret) < 32 {
		return Config{}, fmt.Errorf("auth.signing_secret must contain at least 32 characters")
	}
	if cfg.Auth.CookieName == "" {
		cfg.Auth.CookieName = "agent_session"
	}
	if cfg.PythonAgent.Address == "" {
		return Config{}, fmt.Errorf("python_agent.address is required")
	}
	if cfg.Redis.Address == "" {
		return Config{}, fmt.Errorf("redis.address is required")
	}
	if cfg.Redis.StateTTL == "" {
		cfg.Redis.StateTTL = "30m"
	}
	if cfg.Memory.RecallLimit <= 0 {
		cfg.Memory.RecallLimit = 5
	}
	if cfg.MySQL.MaxOpenConnections <= 0 {
		cfg.MySQL.MaxOpenConnections = 20
	}
	if cfg.MySQL.MaxIdleConnections <= 0 {
		cfg.MySQL.MaxIdleConnections = 5
	}
	for name, value := range map[string]string{"mysql.connection_max_lifetime": cfg.MySQL.ConnectionMaxLifetime, "elasticsearch.timeout": cfg.Elasticsearch.Timeout, "auth.ttl": cfg.Auth.TTL, "python_agent.timeout": cfg.PythonAgent.Timeout, "redis.timeout": cfg.Redis.Timeout, "redis.state_ttl": cfg.Redis.StateTTL} {
		if value != "" {
			if _, err := time.ParseDuration(value); err != nil {
				return Config{}, fmt.Errorf("invalid %s: %w", name, err)
			}
		}
	}
	return cfg, nil
}

func Duration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	parsed, _ := time.ParseDuration(value)
	return parsed
}
