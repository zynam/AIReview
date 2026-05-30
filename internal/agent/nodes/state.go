package nodes

import (
	"aireview/internal/review"
	"aireview/internal/reviewcontext"
)

type Metrics struct {
	Model              string
	FileCount          int
	RuleFindingCount   int
	ContextChunks      int
	KeptChunks         int
	SkippedFiles       int
	PromptTokensApprox int
	DurationMillis     int64
}

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
