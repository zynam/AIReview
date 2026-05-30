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

	minSeverity, err := minSeverity(req)
	if err != nil {
		return review.ReviewReport{}, err
	}
	minConfidence := minConfidence(req)

	ruleHints := review.RuleAnalyzer{}.Analyze(pr)
	aiResponse, err := s.LLM.Review(ctx, llm.ReviewRequest{
		PullRequest:  pr,
		RuleFindings: ruleHints,
		Config:       req.Config,
	})
	if err != nil {
		return review.ReviewReport{}, err
	}

	report := aiResponse.Report
	report.SkippedFiles = append(report.SkippedFiles, skippedFiles...)
	report.Findings = review.FilterFindings(report.Findings, minSeverity, minConfidence)
	review.SortFindings(report.Findings)
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
