package llm

import (
	"context"

	"aireview/internal/config"
	"aireview/internal/review"
)

type Provider interface {
	Review(ctx context.Context, req ReviewRequest) (ReviewResponse, error)
}

type ReviewRequest struct {
	PullRequest  review.PullRequest
	RuleFindings []review.Finding
	Config       config.Config
}

type ReviewResponse struct {
	Report review.ReviewReport
}
