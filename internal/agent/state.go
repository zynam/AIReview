package agent

import (
	"aireview/internal/review"
	"aireview/internal/reviewcontext"
)

type State struct {
	PullRequest       review.PullRequest
	RuleFindings      []review.Finding
	ReviewContext     reviewcontext.ReviewContext
	CompressedContext reviewcontext.ReviewContext
	SystemPrompt      string
	UserPrompt        string
	RawModelOutput    string
	AIReport          review.ReviewReport
	FinalReport       review.ReviewReport
	Metrics           Metrics
}
