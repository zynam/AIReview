package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aireview/internal/config"
	"aireview/internal/review"
)

func TestEinoReviewAgentRunsWorkflowAndMergesRuleFindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q, want /chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "test-model" {
			t.Fatalf("Model = %q", req.Model)
		}
		if len(req.Messages) != 2 || !strings.Contains(req.Messages[1].Content, "context_chunks") {
			t.Fatalf("unexpected messages: %#v", req.Messages)
		}

		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "{\"summary\":\"AI summary\",\"impact\":[\"runtime behavior\"],\"findings\":[{\"severity\":\"high\",\"confidence\":0.9,\"category\":\"security\",\"file\":\"internal/auth.go\",\"line\":10,\"title\":\"AI finding\",\"evidence\":\"token literal\",\"suggestion\":\"use env\",\"needs_human_check\":false}],\"test_assessment\":\"add tests\",\"skipped_files\":[]}"
				}
			}]
		}`))
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	cfg.LLM.BaseURL = server.URL
	cfg.LLM.APIKey = "test-key"
	cfg.LLM.Model = "test-model"
	cfg.Agent.MaxContextTokens = 4000
	cfg.Agent.MaxFilePatchBytes = 4000
	cfg.Agent.MaxFiles = 10

	sink := &recordingSink{}
	result, err := NewEinoReviewAgent(WithHTTPClient(server.Client())).Analyze(context.Background(), AnalyzeRequest{
		PullRequest: review.PullRequest{
			Owner:  "openai",
			Repo:   "openai-go",
			Number: 123,
			Title:  "add auth",
			Files: []review.ChangedFile{{
				Path:      "internal/auth.go",
				Additions: 1,
				Patch:     "@@ -10,0 +10,1 @@\n+const token = \"dev\"",
			}},
		},
		RuleFindings: []review.Finding{{
			Severity:   review.SeverityMedium,
			Confidence: 0.8,
			Category:   review.CategorySecurity,
			File:       "internal/auth.go",
			Line:       10,
			Title:      "Rule finding",
			Evidence:   "rule evidence",
			Suggestion: "rule suggestion",
		}},
		Config:    cfg,
		EventSink: sink,
	})
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if result.Report.Summary != "AI summary" {
		t.Fatalf("Summary = %q", result.Report.Summary)
	}
	if len(result.Report.Findings) != 2 {
		t.Fatalf("findings = %#v, want AI and rule findings", result.Report.Findings)
	}
	if result.Metrics.Model != "test-model" {
		t.Fatalf("Metrics.Model = %q", result.Metrics.Model)
	}
	if result.Metrics.ContextChunks == 0 || result.Metrics.KeptChunks == 0 || result.Metrics.PromptTokensApprox == 0 {
		t.Fatalf("incomplete metrics: %#v", result.Metrics)
	}
	if !sink.has(EventContextBuildStarted) || !sink.has(EventLLMCallCompleted) || !sink.has(EventReviewCompleted) {
		t.Fatalf("events = %#v, want build, llm completed and review completed", sink.events)
	}
}

type recordingSink struct {
	events []Event
}

func (s *recordingSink) Emit(ctx context.Context, event Event) error {
	s.events = append(s.events, event)
	return nil
}

func (s *recordingSink) has(eventType string) bool {
	for _, event := range s.events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
