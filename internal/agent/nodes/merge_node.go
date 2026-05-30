package nodes

import (
	"context"

	"aireview/internal/review"
)

type MergeNode struct{}

func (n MergeNode) Run(ctx context.Context, state State) (State, error) {
	report := review.MergeReports(state.AIReport, state.RuleFindings)
	report.SkippedFiles = append(report.SkippedFiles, state.CompressedContext.SkippedFiles...)
	review.SortFindings(report.Findings)
	state.FinalReport = report
	return state, nil
}
