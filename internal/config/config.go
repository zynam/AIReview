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
	LLM                    LLMConfig     `toml:"llm"`
}

type PublishConfig struct {
	Enabled bool `toml:"enabled"`
}

type LLMConfig struct {
	BaseURL   string `toml:"base_url"`
	APIKeyEnv string `toml:"api_key_env"`
	Model     string `toml:"model"`
}

func DefaultConfig() Config {
	return Config{
		Language: "zh-CN",
		LLM: LLMConfig{
			BaseURL:   "https://api.deepseek.com",
			APIKeyEnv: "LLM_API_KEY",
			Model:     "deepseek-chat",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	applyEnvFallbacks(&cfg)

	if strings.TrimSpace(path) == "" {
		path = DefaultConfigPath
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		} else if err != nil {
			return Config{}, fmt.Errorf("stat config file %s: %w", path, err)
		}
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("load config file %s: %w", path, err)
	}
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
	if apiKeyEnv := strings.TrimSpace(os.Getenv("LLM_API_KEY_ENV")); apiKeyEnv != "" {
		cfg.LLM.APIKeyEnv = apiKeyEnv
	}
}

func normalize(cfg *Config) {
	cfg.Language = strings.TrimSpace(cfg.Language)
	cfg.MinSeverityToPublish = strings.TrimSpace(cfg.MinSeverityToPublish)
	cfg.LLM.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.LLM.BaseURL), "/")
	cfg.LLM.APIKeyEnv = strings.TrimSpace(cfg.LLM.APIKeyEnv)
	cfg.LLM.Model = strings.TrimSpace(cfg.LLM.Model)

	if cfg.Language == "" {
		cfg.Language = DefaultConfig().Language
	}
	if cfg.LLM.BaseURL == "" {
		cfg.LLM.BaseURL = DefaultConfig().LLM.BaseURL
	}
	if cfg.LLM.APIKeyEnv == "" {
		cfg.LLM.APIKeyEnv = DefaultConfig().LLM.APIKeyEnv
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = DefaultConfig().LLM.Model
	}
}
