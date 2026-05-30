package reviewcontext

import "aireview/internal/review"

type ChunkKind string

const (
	ChunkPRMeta      ChunkKind = "pr_meta"
	ChunkCommit      ChunkKind = "commit"
	ChunkPatch       ChunkKind = "patch"
	ChunkRuleFinding ChunkKind = "rule_finding"
	ChunkSummary     ChunkKind = "summary"
)

type Chunk struct {
	ID      string
	File    string
	Kind    ChunkKind
	Content string
	Tokens  int
	Score   float64
}

type ReviewContext struct {
	PullRequest  review.PullRequest
	RuleFindings []review.Finding
	Chunks       []Chunk
	SkippedFiles []string
}
