package nodes

import (
	"context"
	"fmt"

	"aireview/internal/config"
	"aireview/internal/reviewcontext"
)

type ContextBuildNode struct {
	Config config.Config
}

func (n ContextBuildNode) Run(ctx context.Context, state State) (State, error) {
	state.ReviewContext = reviewcontext.Builder{Config: n.Config}.Build(state.PullRequest, state.RuleFindings)
	state.Metrics.FileCount = len(state.PullRequest.Files)
	state.Metrics.RuleFindingCount = len(state.RuleFindings)
	state.Metrics.ContextChunks = len(state.ReviewContext.Chunks)
	return state, nil
}

func (n ContextBuildNode) Message(state State) string {
	return fmt.Sprintf("built %d context chunks", len(state.ReviewContext.Chunks))
}
