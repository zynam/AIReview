package agent

import (
	"context"

	"aireview/internal/config"
	"aireview/internal/review"
)

type ReviewAgent interface {
	Analyze(ctx context.Context, req AnalyzeRequest) (AnalyzeResult, error)
}

type AnalyzeRequest struct {
	PullRequest  review.PullRequest
	RuleFindings []review.Finding
	Config       config.Config
	EventSink    EventSink
}

type AnalyzeResult struct {
	Report  review.ReviewReport
	Metrics Metrics
}
