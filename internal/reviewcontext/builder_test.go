package reviewcontext

import (
	"testing"

	"aireview/internal/config"
	"aireview/internal/diff"
	"aireview/internal/review"
)

func TestBuilderCreatesPatchChunkForEachChangedFile(t *testing.T) {
	pr := review.PullRequest{
		Owner:  "acme",
		Repo:   "api",
		Number: 7,
		Title:  "add handler",
		Files: []review.ChangedFile{
			{Path: "internal/server/handler.go", FileKind: diff.FileKindSource, Additions: 2, Patch: "@@ -1 +1 @@\n+package server"},
			{Path: "internal/server/handler_test.go", FileKind: diff.FileKindTest, Additions: 3, Patch: "@@ -1 +1 @@\n+package server"},
		},
	}

	ctx := Builder{}.Build(pr, nil)

	patchChunks := map[string]Chunk{}
	for _, chunk := range ctx.Chunks {
		if chunk.Kind == ChunkPatch {
			patchChunks[chunk.File] = chunk
		}
	}
	for _, file := range pr.Files {
		chunk, ok := patchChunks[file.Path]
		if !ok {
			t.Fatalf("missing patch chunk for %s", file.Path)
		}
		if chunk.Tokens == 0 {
			t.Fatalf("patch chunk for %s did not estimate tokens", file.Path)
		}
	}
}

func TestBuilderCreatesRuleFindingChunks(t *testing.T) {
	finding := review.Finding{
		Severity:   review.SeverityHigh,
		Confidence: 0.9,
		Category:   review.CategorySecurity,
		File:       "internal/auth/token.go",
		Line:       42,
		Title:      "token leak",
		Evidence:   "new token literal",
		Suggestion: "move to env",
	}

	ctx := Builder{}.Build(review.PullRequest{}, []review.Finding{finding})

	for _, chunk := range ctx.Chunks {
		if chunk.Kind == ChunkRuleFinding {
			if chunk.File != finding.File {
				t.Fatalf("rule finding chunk file = %q, want %q", chunk.File, finding.File)
			}
			if chunk.Score <= defaultRuleFindingScore {
				t.Fatalf("high severity rule finding score = %.2f, want above %d", chunk.Score, defaultRuleFindingScore)
			}
			return
		}
	}
	t.Fatal("missing rule finding chunk")
}

func TestBuilderScoresHighRiskSourceAboveGeneratedFile(t *testing.T) {
	cfg := config.Config{HighRiskPaths: []string{"internal/auth/*"}}
	pr := review.PullRequest{
		Files: []review.ChangedFile{
			{Path: "internal/auth/token.go", FileKind: diff.FileKindSource, Patch: "@@ -1 +1 @@\n+func token() {}"},
			{Path: "web/dist/app.generated.js", FileKind: diff.FileKindGenerated, Patch: "@@ -1 +1 @@\n+bundle"},
		},
	}

	ctx := Builder{Config: cfg}.Build(pr, nil)

	scores := map[string]float64{}
	for _, chunk := range ctx.Chunks {
		if chunk.Kind == ChunkPatch {
			scores[chunk.File] = chunk.Score
		}
	}
	if scores["internal/auth/token.go"] <= scores["web/dist/app.generated.js"] {
		t.Fatalf("high risk source score %.2f should exceed generated score %.2f", scores["internal/auth/token.go"], scores["web/dist/app.generated.js"])
	}
}
