package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

const DefaultConfigPath = ".aireview.toml"

type Config struct {
	Language               string        `toml:"language"`
	MinSeverityToPublish   string        `toml:"min_severity_to_publish"`
	MinConfidenceToPublish float64       `toml:"min_confidence_to_publish"`
	IgnorePaths            []string      `toml:"ignore_paths"`
	HighRiskPaths          []string      `toml:"high_risk_paths"`
	ReviewFocus            []string      `toml:"review_focus"`
	Publish                PublishConfig `toml:"publish"`
	Agent                  AgentConfig   `toml:"agent"`
	LLM                    LLMConfig     `toml:"llm"`
	MySQL                  MySQLConfig   `toml:"mysql"`
}

type PublishConfig struct {
	Enabled bool `toml:"enabled"`
}

type AgentConfig struct {
	MaxContextTokens  int `toml:"max_context_tokens"`
	MaxFilePatchBytes int `toml:"max_file_patch_bytes"`
	MaxFiles          int `toml:"max_files"`
	TimeoutSeconds    int `toml:"timeout_seconds"`
}

type LLMConfig struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
}

type MySQLConfig struct {
	DSN string `toml:"dsn"`
}

func DefaultConfig() Config {
	return Config{
		Language: "zh-CN",
		Agent: AgentConfig{
			MaxContextTokens:  12000,
			MaxFilePatchBytes: 20000,
			MaxFiles:          50,
			TimeoutSeconds:    180,
		},
		LLM: LLMConfig{
			BaseURL: "https://api.deepseek.com",
			Model:   "deepseek-chat",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()

	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			applyEnvFallbacks(&cfg)
			normalize(&cfg)
			return cfg, nil
		} else if err != nil {
			return Config{}, fmt.Errorf("stat config file %s: %w", path, err)
		}
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("load config file %s: %w", path, err)
	}
	normalize(&cfg)
	applyEnvFallbacks(&cfg)
	normalize(&cfg)
	return cfg, nil
}

func applyEnvFallbacks(cfg *Config) {
	if baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL")); baseURL != "" {
		cfg.LLM.BaseURL = baseURL
	}
	if model := strings.TrimSpace(os.Getenv("LLM_MODEL")); model != "" {
		cfg.LLM.Model = model
	}
}

func normalize(cfg *Config) {
	cfg.Language = strings.TrimSpace(cfg.Language)
	cfg.MinSeverityToPublish = strings.TrimSpace(cfg.MinSeverityToPublish)
	cfg.LLM.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.LLM.BaseURL), "/")
	cfg.LLM.APIKey = strings.TrimSpace(cfg.LLM.APIKey)
	cfg.LLM.Model = strings.TrimSpace(cfg.LLM.Model)
	cfg.MySQL.DSN = strings.TrimSpace(cfg.MySQL.DSN)

	if cfg.Language == "" {
		cfg.Language = DefaultConfig().Language
	}
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = DefaultConfig().LLM.BaseURL
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = DefaultConfig().LLM.Model
	}
	defaults := DefaultConfig().Agent
	if cfg.Agent.MaxContextTokens <= 0 {
		cfg.Agent.MaxContextTokens = defaults.MaxContextTokens
	}
	if cfg.Agent.MaxFilePatchBytes <= 0 {
		cfg.Agent.MaxFilePatchBytes = defaults.MaxFilePatchBytes
	}
	if cfg.Agent.MaxFiles <= 0 {
		cfg.Agent.MaxFiles = defaults.MaxFiles
	}
	if cfg.Agent.TimeoutSeconds <= 0 {
		cfg.Agent.TimeoutSeconds = defaults.TimeoutSeconds
	}
}
