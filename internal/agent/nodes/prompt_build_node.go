package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aireview/internal/config"
	"aireview/internal/review"
	"aireview/internal/reviewcontext"
)

const systemPrompt = `You are an AI pull request review assistant.
Only review the PR metadata, commits, patches, rule findings, and context chunks provided by the user.
Do not invent files, line numbers, behavior, test results, or repository context.
Return only a valid JSON object matching the requested ReviewReport schema.`

type PromptBuildNode struct {
	Config config.Config
}

func (n PromptBuildNode) Run(ctx context.Context, state State) (State, error) {
	payload := promptPayload{
		Language:     n.Config.Language,
		ReviewFocus:  n.Config.ReviewFocus,
		PullRequest:  state.PullRequest,
		Commits:      state.PullRequest.Commits,
		FileStats:    fileStats(state.PullRequest.Files),
		Context:      state.CompressedContext.Chunks,
		RuleFindings: state.RuleFindings,
		OutputSchema: map[string]any{
			"summary":         "string",
			"impact":          []string{"string"},
			"findings":        []string{"Finding objects"},
			"test_assessment": "string",
			"skipped_files":   []string{"string"},
		},
		FindingSchema: review.Finding{},
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return State{}, fmt.Errorf("build prompt payload: %w", err)
	}

	state.SystemPrompt = systemPrompt
	state.UserPrompt = strings.Join([]string{
		"Analyze this GitHub pull request and return a ReviewReport JSON object.",
		"Use the requested language for natural-language fields.",
		"Every finding must be traceable to the provided patch or context chunk.",
		string(body),
	}, "\n\n")
	state.Metrics.PromptTokensApprox = reviewcontext.EstimateTokens(state.SystemPrompt + "\n" + state.UserPrompt)
	return state, nil
}

type promptPayload struct {
	Language      string                `json:"language"`
	ReviewFocus   []string              `json:"review_focus,omitempty"`
	PullRequest   review.PullRequest    `json:"pull_request"`
	Commits       []review.Commit       `json:"commits"`
	FileStats     []fileStat            `json:"file_stats"`
	Context       []reviewcontext.Chunk `json:"context_chunks"`
	RuleFindings  []review.Finding      `json:"rule_findings"`
	OutputSchema  map[string]any        `json:"output_schema"`
	FindingSchema review.Finding        `json:"finding_schema"`
}

type fileStat struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Language  string `json:"language,omitempty"`
	FileKind  string `json:"file_kind,omitempty"`
}

func fileStats(files []review.ChangedFile) []fileStat {
	stats := make([]fileStat, 0, len(files))
	for _, file := range files {
		stats = append(stats, fileStat{
			Path:      file.Path,
			Status:    file.Status,
			Additions: file.Additions,
			Deletions: file.Deletions,
			Language:  file.Language,
			FileKind:  file.FileKind,
		})
	}
	return stats
}
