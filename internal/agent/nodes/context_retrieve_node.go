package nodes

import (
	"context"
	"fmt"

	"aireview/internal/reviewcontext"
)

type ContextRetrieveNode struct{}

func (n ContextRetrieveNode) Run(ctx context.Context, state State) (State, error) {
	state.ReviewContext = reviewcontext.Retriever{}.Retrieve(state.ReviewContext)
	return state, nil
}

func (n ContextRetrieveNode) Message(state State) string {
	return fmt.Sprintf("ranked %d context chunks", len(state.ReviewContext.Chunks))
}
