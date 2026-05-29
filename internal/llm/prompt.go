package llm

import (
	"encoding/json"
	"fmt"
	"strings"

	"aireview/internal/review"
)

const systemPrompt = `You are an AI pull request reviewer.
Review only the supplied pull request metadata, patches, and rule findings.
Return JSON only, with no Markdown wrapper.
Every finding must include file, line, evidence, suggestion, severity, confidence, category, title, and needs_human_check.
If evidence is uncertain, lower confidence and set needs_human_check=true.`

func buildPrompt(req ReviewRequest) (string, error) {
	payload := promptPayload{
		Language:     req.Config.Language,
		ReviewFocus:  req.Config.ReviewFocus,
		PullRequest:  req.PullRequest,
		RuleFindings: req.RuleFindings,
		OutputSchema: map[string]any{
			"summary":         "string",
			"impact":          []string{"string"},
			"findings":        []string{"Finding objects"},
			"test_assessment": "string",
			"skipped_files":   []string{"string"},
		},
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build LLM prompt payload: %w", err)
	}

	return strings.Join([]string{
		"Analyze this GitHub pull request and return a ReviewReport JSON object.",
		"Use the requested language for prose fields when possible.",
		"Do not invent files, line numbers, behavior, or test results beyond the supplied patch context.",
		string(body),
	}, "\n\n"), nil
}

type promptPayload struct {
	Language     string             `json:"language"`
	ReviewFocus  []string           `json:"review_focus,omitempty"`
	PullRequest  review.PullRequest `json:"pull_request"`
	RuleFindings []review.Finding   `json:"rule_findings"`
	OutputSchema map[string]any     `json:"output_schema"`
}
