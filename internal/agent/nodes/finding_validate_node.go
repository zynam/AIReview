package nodes

import (
	"context"

	"aireview/internal/review"
)

type FindingValidateNode struct {
	Validate func(review.PullRequest, review.ReviewReport) review.ReviewReport
}

func (n FindingValidateNode) Run(ctx context.Context, state State) (State, error) {
	validate := n.Validate
	if validate == nil {
		validate = func(_ review.PullRequest, report review.ReviewReport) review.ReviewReport {
			return report
		}
	}
	state.AIReport = validate(state.PullRequest, state.AIReport)
	return state, nil
}
