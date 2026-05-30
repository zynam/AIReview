package reviewcontext

import "aireview/internal/config"

type TokenBudget struct {
	MaxContextTokens  int
	MaxFilePatchBytes int
	MaxFiles          int
}

func NewTokenBudget(cfg config.Config) TokenBudget {
	return TokenBudget{
		MaxContextTokens:  cfg.Agent.MaxContextTokens,
		MaxFilePatchBytes: cfg.Agent.MaxFilePatchBytes,
		MaxFiles:          cfg.Agent.MaxFiles,
	}
}

func (b TokenBudget) normalize() TokenBudget {
	if b.MaxContextTokens <= 0 {
		b.MaxContextTokens = 12000
	}
	if b.MaxFilePatchBytes <= 0 {
		b.MaxFilePatchBytes = 20000
	}
	if b.MaxFiles <= 0 {
		b.MaxFiles = 50
	}
	return b
}
