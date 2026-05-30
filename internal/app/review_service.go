package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"aireview/internal/agent"
	"aireview/internal/config"
	"aireview/internal/diff"
	"aireview/internal/github"
	"aireview/internal/review"
)

type ReviewService struct {
	GitHub github.Client
	Agent  agent.ReviewAgent
}

type ReviewPRRequest struct {
	Ref           github.PRRef
	Config        config.Config
	MinSeverity   string
	MinConfidence float64
	MaxFiles      int
	EventSink     agent.EventSink
}

type ReviewPRResult struct {
	Report      review.ReviewReport
	PullRequest review.PullRequest
	Agent       agent.AnalyzeResult
}

func (s *ReviewService) ReviewPR(ctx context.Context, req ReviewPRRequest) (review.ReviewReport, error) {
	result, err := s.ReviewPRWithDetails(ctx, req)
	return result.Report, err
}

func (s *ReviewService) ReviewPRWithDetails(ctx context.Context, req ReviewPRRequest) (ReviewPRResult, error) {
	if s.GitHub == nil {
		return ReviewPRResult{}, fmt.Errorf("github client is required")
	}
	if s.Agent == nil {
		return ReviewPRResult{}, fmt.Errorf("review agent is required")
	}

	pr, err := s.GitHub.GetPullRequest(ctx, req.Ref)
	if err != nil {
		return ReviewPRResult{}, err
	}
	classifyFiles(pr.Files)
	pr, ignoredFiles := ignoreFiles(pr, req.Config.IgnorePaths)
	pr, skippedFiles := limitFiles(pr, req.MaxFiles)
	skippedFiles = append(skippedFiles, ignoredFiles...)

	minSeverity, err := minSeverity(req)
	if err != nil {
		return ReviewPRResult{PullRequest: pr}, err
	}
	minConfidence := minConfidence(req)

	ruleHints := review.RuleAnalyzer{}.Analyze(pr)
	agentResult, err := s.Agent.Analyze(ctx, agent.AnalyzeRequest{
		PullRequest:  pr,
		RuleFindings: ruleHints,
		Config:       req.Config,
		EventSink:    req.EventSink,
	})
	if err != nil {
		return ReviewPRResult{PullRequest: pr}, err
	}

	report := agentResult.Report
	report.SkippedFiles = append(report.SkippedFiles, skippedFiles...)
	report.Findings = review.FilterFindings(report.Findings, minSeverity, minConfidence)
	review.SortFindings(report.Findings)
	return ReviewPRResult{
		Report:      report,
		PullRequest: pr,
		Agent:       agentResult,
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

func ignoreFiles(pr review.PullRequest, patterns []string) (review.PullRequest, []string) {
	if len(patterns) == 0 || len(pr.Files) == 0 {
		return pr, nil
	}

	files := make([]review.ChangedFile, 0, len(pr.Files))
	ignored := make([]string, 0)
	for _, file := range pr.Files {
		if pathMatchesAny(file.Path, patterns) {
			ignored = append(ignored, file.Path)
			continue
		}
		files = append(files, file)
	}
	pr.Files = files
	return pr, ignored
}

func pathMatchesAny(path string, patterns []string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(strings.ReplaceAll(pattern, "\\", "/"))
		if pattern == "" {
			continue
		}
		if ok, _ := filepath.Match(pattern, normalized); ok {
			return true
		}
		trimmed := strings.Trim(pattern, "*")
		if trimmed != "" && strings.Contains(normalized, trimmed) {
			return true
		}
	}
	return false
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
