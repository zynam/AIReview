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

type ReviewPRResult struct {
	Report      review.ReviewReport
	PullRequest review.PullRequest
}

func (s *ReviewService) ReviewPR(ctx context.Context, req ReviewPRRequest) (review.ReviewReport, error) {
	result, err := s.ReviewPRWithDetails(ctx, req)
	return result.Report, err
}

func (s *ReviewService) ReviewPRWithDetails(ctx context.Context, req ReviewPRRequest) (ReviewPRResult, error) {
	if s.GitHub == nil {
		return ReviewPRResult{}, fmt.Errorf("github client is required")
	}
	if s.LLM == nil {
		return ReviewPRResult{}, fmt.Errorf("LLM provider is required")
	}

	pr, err := s.GitHub.GetPullRequest(ctx, req.Ref)
	if err != nil {
		return ReviewPRResult{}, err
	}
	classifyFiles(pr.Files)
	pr, skippedFiles := limitFiles(pr, req.MaxFiles)

	minSeverity, err := minSeverity(req)
	if err != nil {
		return ReviewPRResult{PullRequest: pr}, err
	}
	minConfidence := minConfidence(req)

	ruleHints := review.RuleAnalyzer{}.Analyze(pr)
	aiResponse, err := s.LLM.Review(ctx, llm.ReviewRequest{
		PullRequest:  pr,
		RuleFindings: ruleHints,
		Config:       req.Config,
	})
	if err != nil {
		return ReviewPRResult{PullRequest: pr}, err
	}

	report := aiResponse.Report
	report.SkippedFiles = append(report.SkippedFiles, skippedFiles...)
	report.Findings = review.FilterFindings(report.Findings, minSeverity, minConfidence)
	review.SortFindings(report.Findings)
	return ReviewPRResult{
		Report:      report,
		PullRequest: pr,
	}, nil
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
