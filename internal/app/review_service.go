package app

import (
	"context"
	"fmt"

	"aireview/internal/config"
	"aireview/internal/diff"
	"aireview/internal/github"
	"aireview/internal/llm"
	"aireview/internal/review"
)

type ReviewService struct {
	GitHub github.Client
	LLM    llm.Provider
}

type ReviewPRRequest struct {
	Ref           github.PRRef
	Config        config.Config
	MinSeverity   string
	MinConfidence float64
	MaxFiles      int
}

func (s *ReviewService) ReviewPR(ctx context.Context, req ReviewPRRequest) (review.ReviewReport, error) {
	if s.GitHub == nil {
		return review.ReviewReport{}, fmt.Errorf("github client is required")
	}
	if s.LLM == nil {
		return review.ReviewReport{}, fmt.Errorf("LLM provider is required")
	}

	pr, err := s.GitHub.GetPullRequest(ctx, req.Ref)
	if err != nil {
		return review.ReviewReport{}, err
	}
	classifyFiles(pr.Files)
	pr, skippedFiles := limitFiles(pr, req.MaxFiles)

	ruleFindings := review.RuleAnalyzer{}.Analyze(pr)
	aiResponse, llmErr := s.LLM.Review(ctx, llm.ReviewRequest{
		PullRequest:  pr,
		RuleFindings: ruleFindings,
		Config:       req.Config,
	})

	report := review.MergeReports(aiResponse.Report, ruleFindings)
	if report.Summary == "" {
		report.Summary = fallbackSummary(req.Ref, llmErr)
	}
	report.SkippedFiles = append(report.SkippedFiles, skippedFiles...)

	minSeverity, err := minSeverity(req)
	if err != nil {
		return report, err
	}
	minConfidence := minConfidence(req)
	report.Findings = review.FilterFindings(report.Findings, minSeverity, minConfidence)
	review.SortFindings(report.Findings)

	if llmErr != nil {
		return report, llmErr
	}
	return report, nil
}

func classifyFiles(files []review.ChangedFile) {
	for i := range files {
		classification := diff.ClassifyFile(files[i].Path)
		files[i].Language = classification.Language
		files[i].FileKind = classification.FileKind
	}
}

func limitFiles(pr review.PullRequest, maxFiles int) (review.PullRequest, []string) {
	if maxFiles <= 0 || len(pr.Files) <= maxFiles {
		return pr, nil
	}

	skipped := make([]string, 0, len(pr.Files)-maxFiles)
	for _, file := range pr.Files[maxFiles:] {
		skipped = append(skipped, file.Path)
	}
	pr.Files = pr.Files[:maxFiles]
	return pr, skipped
}

func minSeverity(req ReviewPRRequest) (review.Severity, error) {
	value := req.MinSeverity
	if value == "" {
		value = req.Config.MinSeverityToPublish
	}
	severity, ok := review.ParseSeverity(value)
	if !ok {
		return "", fmt.Errorf("invalid minimum severity %q", value)
	}
	return severity, nil
}

func minConfidence(req ReviewPRRequest) float64 {
	if req.MinConfidence > 0 {
		return req.MinConfidence
	}
	return req.Config.MinConfidenceToPublish
}

func fallbackSummary(ref github.PRRef, llmErr error) string {
	if llmErr != nil {
		return fmt.Sprintf("AI review failed for %s/%s#%d; showing rule-based findings only.", ref.Owner, ref.Repo, ref.Number)
	}
	return fmt.Sprintf("Review completed for %s/%s#%d.", ref.Owner, ref.Repo, ref.Number)
}
