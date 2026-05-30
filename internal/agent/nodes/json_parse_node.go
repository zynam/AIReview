package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aireview/internal/review"
)

type JSONParseNode struct{}

func (n JSONParseNode) Run(ctx context.Context, state State) (State, error) {
	cleaned := stripJSONFence(state.RawModelOutput)
	var report review.ReviewReport
	if err := json.Unmarshal([]byte(cleaned), &report); err != nil {
		return State{}, fmt.Errorf("LLM returned non-JSON ReviewReport: %w; response excerpt: %s", err, excerpt(state.RawModelOutput, 500))
	}
	state.AIReport = report
	return state, nil
}

func stripJSONFence(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "```") {
		return content
	}
	lines := strings.Split(content, "\n")
	if len(lines) >= 3 && strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "```") {
		return strings.TrimSpace(strings.Join(lines[1:len(lines)-1], "\n"))
	}
	return content
}

func excerpt(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "..."
}
