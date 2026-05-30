package nodes

import (
	"context"
	"fmt"

	"aireview/internal/config"
	"aireview/internal/reviewcontext"
)

type ContextCompressNode struct {
	Config config.Config
}

func (n ContextCompressNode) Run(ctx context.Context, state State) (State, error) {
	state.CompressedContext = reviewcontext.Compressor{
		Budget: reviewcontext.NewTokenBudget(n.Config),
	}.Compress(state.ReviewContext)
	state.Metrics.KeptChunks = len(state.CompressedContext.Chunks)
	state.Metrics.SkippedFiles = len(state.CompressedContext.SkippedFiles)
	return state, nil
}

func (n ContextCompressNode) Message(state State) string {
	return fmt.Sprintf("kept %d chunks, skipped %d files", len(state.CompressedContext.Chunks), len(state.CompressedContext.SkippedFiles))
}
