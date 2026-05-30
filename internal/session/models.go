package session

import (
	"time"

	"aireview/internal/agent"
	"aireview/internal/review"
)

type Status string

const (
	StatusCreated         Status = "created"
	StatusFetchingPR      Status = "fetching_pr"
	StatusBuildingContext Status = "building_context"
	StatusAnalyzing       Status = "analyzing"
	StatusCompleted       Status = "completed"
	StatusFailed          Status = "failed"
	StatusCancelled       Status = "cancelled"
)

type ReviewSession struct {
	ID             string    `json:"id"`
	ReviewerID     string    `json:"reviewer_id"`
	Owner          string    `json:"owner"`
	Repo           string    `json:"repo"`
	PRNumber       int       `json:"pr_number"`
	HeadSHA        string    `json:"head_sha"`
	Status         Status    `json:"status"`
	Summary        string    `json:"summary"`
	Impact         []string  `json:"impact"`
	TestAssessment string    `json:"test_assessment"`
	SkippedFiles   []string  `json:"skipped_files,omitempty"`
	Error          string    `json:"error,omitempty"`
	FindingsCount  int64     `json:"findings_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ReviewDetail struct {
	ReviewSession
	Findings []review.Finding `json:"findings"`
	LLMCalls []LLMCall        `json:"llm_calls,omitempty"`
}

type ContextChunk struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	File      string    `json:"file"`
	Kind      string    `json:"kind"`
	Content   string    `json:"content"`
	Tokens    int       `json:"tokens"`
	Score     float64   `json:"score"`
	CreatedAt time.Time `json:"created_at"`
}

type ReviewEvent struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type LLMCall struct {
	ID                 string    `json:"id"`
	SessionID          string    `json:"session_id"`
	Model              string    `json:"model"`
	FileCount          int       `json:"file_count"`
	RuleFindingCount   int       `json:"rule_finding_count"`
	ContextChunks      int       `json:"context_chunks"`
	KeptChunks         int       `json:"kept_chunks"`
	SkippedFiles       int       `json:"skipped_files"`
	PromptTokensApprox int       `json:"prompt_tokens_approx"`
	DurationMillis     int64     `json:"duration_millis"`
	CreatedAt          time.Time `json:"created_at"`
}

type AgentArtifacts struct {
	ContextChunks []ContextChunk
	Metrics       agent.Metrics
}

type ListFilter struct {
	ReviewerID string
	Owner      string
	Repo       string
	Status     Status
}
