package reviewcontext

import (
	"testing"

	"aireview/internal/review"
)

func TestRetrieverPrioritizesHighSeverityRuleFindingFiles(t *testing.T) {
	ctx := ReviewContext{
		RuleFindings: []review.Finding{
			{Severity: review.SeverityHigh, File: "docs/readme.md", Title: "risky docs change"},
		},
		Chunks: []Chunk{
			{ID: "source", File: "internal/service.go", Kind: ChunkPatch, Score: 100},
			{ID: "docs", File: "docs/readme.md", Kind: ChunkPatch, Score: 10},
		},
	}

	got := Retriever{}.Retrieve(ctx)

	if got.Chunks[0].File != "docs/readme.md" {
		t.Fatalf("first chunk file = %q, want high severity finding file", got.Chunks[0].File)
	}
}
