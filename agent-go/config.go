package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Config struct {
	Elasticsearch ElasticsearchConfig `json:"elasticsearch"`
	PythonAgent   PythonAgentConfig   `json:"python_agent"`
	Memory        MemoryConfig        `json:"memory"`
}

type ElasticsearchConfig struct {
	Addresses []string `json:"addresses"`
	Username  string   `json:"username"`
	Password  string   `json:"password"`
	APIKey    string   `json:"api_key"`
	Index     string   `json:"index"`
	Timeout   string   `json:"timeout"`
}

type PythonAgentConfig struct {
	Address string `json:"address"`
	Timeout string `json:"timeout"`
}

type MemoryConfig struct {
	RecallLimit int `json:"recall_limit"`
}

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %s: %w", path, err)
	}
	if len(cfg.Elasticsearch.Addresses) == 0 || cfg.Elasticsearch.Index == "" {
		return Config{}, fmt.Errorf("elasticsearch.addresses and elasticsearch.index are required")
	}
	if cfg.PythonAgent.Address == "" {
		return Config{}, fmt.Errorf("python_agent.address is required")
	}
	if cfg.Memory.RecallLimit <= 0 {
		cfg.Memory.RecallLimit = 5
	}
	if _, err := parseDuration(cfg.Elasticsearch.Timeout, 10*time.Second); err != nil {
		return Config{}, fmt.Errorf("invalid elasticsearch.timeout: %w", err)
	}
	if _, err := parseDuration(cfg.PythonAgent.Timeout, 60*time.Second); err != nil {
		return Config{}, fmt.Errorf("invalid python_agent.timeout: %w", err)
	}
	return cfg, nil
}

func parseDuration(value string, fallback time.Duration) (time.Duration, error) {
	if value == "" {
		return fallback, nil
	}
	return time.ParseDuration(value)
}
