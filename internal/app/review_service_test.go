package app

import (
	"context"
	"errors"
	"testing"

	"aireview/internal/agent"
	"aireview/internal/config"
	"aireview/internal/github"
	"aireview/internal/review"
)

func TestReviewServiceUsesAIOnlyAndFilters(t *testing.T) {
	reviewAgent := &fakeReviewAgent{report: review.ReviewReport{
		Summary: "AI summary",
		Findings: []review.Finding{{
			Severity:   review.SeverityLow,
			Confidence: 0.95,
			Category:   review.CategorySecurity,
			File:       "internal/auth.go",
			Line:       10,
			Title:      "Filtered low finding",
		}, {
			Severity:   review.SeverityMedium,
			Confidence: 0.9,
			Category:   review.CategorySecurity,
			File:       "internal/auth.go",
			Line:       10,
			Title:      "AI finding",
			Evidence:   "AI evidence",
			Suggestion: "Use managed secret storage.",
		}},
		TestAssessment: "Tests should cover auth.",
	}}
	service := ReviewService{
		GitHub: fakeGitHubClient{pr: review.PullRequest{
			Owner:  "openai",
			Repo:   "openai-go",
			Number: 123,
			Files: []review.ChangedFile{{
				Path:      "internal/auth.go",
				Additions: 1,
				Patch: `@@ -10,1 +10,2 @@
+const token = "dev"`,
			}},
		}},
		Agent: reviewAgent,
	}

	report, err := service.ReviewPR(context.Background(), ReviewPRRequest{
		Ref:           github.PRRef{Owner: "openai", Repo: "openai-go", Number: 123},
		Config:        config.DefaultConfig(),
		MinSeverity:   "medium",
		MinConfidence: 0.8,
	})
	if err != nil {
		t.Fatalf("ReviewPR() error = %v", err)
	}
	if report.Summary != "AI summary" {
		t.Fatalf("Summary = %q", report.Summary)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("len(Findings) = %d, want 1: %#v", len(report.Findings), report.Findings)
	}
	if report.Findings[0].Title != "AI finding" {
		t.Fatalf("Title = %q, want AI finding", report.Findings[0].Title)
	}
	if len(reviewAgent.lastRequest.RuleFindings) == 0 {
		t.Fatal("RuleFindings empty, want local hints passed to AI")
	}
	for _, finding := range reviewAgent.lastRequest.RuleFindings {
		if finding.Title == "Sensitive keyword added" || finding.Title == "Configuration file changed" {
			t.Fatalf("RuleFindings contain English title: %#v", finding)
		}
	}
}

func TestReviewServiceReturnsErrorWithoutRuleFallbackWhenLLMFails(t *testing.T) {
	service := ReviewService{
		GitHub: fakeGitHubClient{pr: review.PullRequest{
			Owner:  "openai",
			Repo:   "openai-go",
			Number: 123,
			Files: []review.ChangedFile{{
				Path:      "go.mod",
				Additions: 1,
			}},
		}},
		Agent: &fakeReviewAgent{err: errors.New("provider unavailable")},
	}

	report, err := service.ReviewPR(context.Background(), ReviewPRRequest{
		Ref:    github.PRRef{Owner: "openai", Repo: "openai-go", Number: 123},
		Config: config.DefaultConfig(),
	})
	if err == nil {
		t.Fatal("ReviewPR() error = nil, want LLM error")
	}
	if report.Summary != "" || len(report.Findings) != 0 {
		t.Fatalf("Report = %#v, want empty report without local fallback", report)
	}
}

func TestReviewServiceMaxFiles(t *testing.T) {
	reviewAgent := &fakeReviewAgent{report: review.ReviewReport{Summary: "ok"}}
	service := ReviewService{
		GitHub: fakeGitHubClient{pr: review.PullRequest{
			Owner:  "openai",
			Repo:   "openai-go",
			Number: 123,
			Files: []review.ChangedFile{
				{Path: "internal/one.go", Additions: 1},
				{Path: "internal/two.go", Additions: 1},
			},
		}},
		Agent: reviewAgent,
	}

	report, err := service.ReviewPR(context.Background(), ReviewPRRequest{
		Ref:      github.PRRef{Owner: "openai", Repo: "openai-go", Number: 123},
		Config:   config.DefaultConfig(),
		MaxFiles: 1,
	})
	if err != nil {
		t.Fatalf("ReviewPR() error = %v", err)
	}
	if len(report.SkippedFiles) != 1 || report.SkippedFiles[0] != "internal/two.go" {
		t.Fatalf("SkippedFiles = %#v", report.SkippedFiles)
	}
	if len(reviewAgent.lastRequest.PullRequest.Files) != 1 {
		t.Fatalf("Agent files = %d, want 1", len(reviewAgent.lastRequest.PullRequest.Files))
	}
}

func TestReviewServiceInvalidSeverity(t *testing.T) {
	service := ReviewService{
		GitHub: fakeGitHubClient{pr: review.PullRequest{Files: []review.ChangedFile{{Path: "internal/one.go"}}}},
		Agent:  &fakeReviewAgent{report: review.ReviewReport{Summary: "ok"}},
	}

	report, err := service.ReviewPR(context.Background(), ReviewPRRequest{
		Ref:         github.PRRef{Owner: "openai", Repo: "openai-go", Number: 123},
		Config:      config.DefaultConfig(),
		MinSeverity: "critical",
	})
	if err == nil {
		t.Fatal("ReviewPR() error = nil, want invalid severity error")
	}
	if report.Summary != "" {
		t.Fatalf("Summary = %q, want empty report", report.Summary)
	}
}

type fakeGitHubClient struct {
	pr  review.PullRequest
	err error
}

func (f fakeGitHubClient) GetPullRequest(ctx context.Context, ref github.PRRef) (review.PullRequest, error) {
	return f.pr, f.err
}

type fakeReviewAgent struct {
	report      review.ReviewReport
	err         error
	lastRequest agent.AnalyzeRequest
}

func (f *fakeReviewAgent) Analyze(ctx context.Context, req agent.AnalyzeRequest) (agent.AnalyzeResult, error) {
	f.lastRequest = req
	return agent.AnalyzeResult{Report: f.report}, f.err
}
