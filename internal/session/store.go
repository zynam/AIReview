package session

import (
	"context"
	"errors"

	"aireview/internal/review"
)

var ErrNotFound = errors.New("session resource not found")

type Store interface {
	CreateSession(ctx context.Context, session ReviewSession) error
	GetSession(ctx context.Context, id string) (ReviewSession, error)
	ListSessions(ctx context.Context, filter ListFilter) ([]ReviewSession, error)
	UpdateStatus(ctx context.Context, id string, status Status, message string) error
	UpdateHeadSHA(ctx context.Context, id string, headSHA string) error
	SaveReport(ctx context.Context, id string, report review.ReviewReport) error
	SaveAgentArtifacts(ctx context.Context, id string, artifacts AgentArtifacts) error
	ListFindings(ctx context.Context, sessionID string) ([]review.Finding, error)
	ListContexts(ctx context.Context, sessionID string) ([]ContextChunk, error)
	ListEvents(ctx context.Context, sessionID string) ([]ReviewEvent, error)
	ListLLMCalls(ctx context.Context, sessionID string) ([]LLMCall, error)
	SaveReviewEvent(ctx context.Context, event ReviewEvent) error
	UpdateFindingFeedback(ctx context.Context, findingID string, status string) error
}
