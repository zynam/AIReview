package reviewcontext

import (
	"strings"
	"testing"
)

func TestCompressorRespectsTokenBudgetAndRecordsSkippedFiles(t *testing.T) {
	ctx := ReviewContext{
		Chunks: []Chunk{
			{ID: "a", File: "a.go", Kind: ChunkPatch, Content: strings.Repeat("a", 80), Tokens: 20, Score: 100},
			{ID: "b", File: "b.go", Kind: ChunkPatch, Content: strings.Repeat("b", 80), Tokens: 20, Score: 90},
			{ID: "c", File: "c.go", Kind: ChunkPatch, Content: strings.Repeat("c", 80), Tokens: 20, Score: 80},
		},
	}

	got := Compressor{Budget: TokenBudget{MaxContextTokens: 45, MaxFilePatchBytes: 1000, MaxFiles: 50}}.Compress(ctx)

	if len(got.Chunks) != 2 {
		t.Fatalf("kept chunks = %d, want 2", len(got.Chunks))
	}
	if len(got.SkippedFiles) != 1 || got.SkippedFiles[0] != "c.go" {
		t.Fatalf("SkippedFiles = %#v, want c.go", got.SkippedFiles)
	}
}

func TestCompressorRespectsMaxFiles(t *testing.T) {
	ctx := ReviewContext{
		Chunks: []Chunk{
			{ID: "a", File: "a.go", Kind: ChunkPatch, Content: "a", Tokens: 1, Score: 100},
			{ID: "b", File: "b.go", Kind: ChunkPatch, Content: "b", Tokens: 1, Score: 90},
		},
	}

	got := Compressor{Budget: TokenBudget{MaxContextTokens: 100, MaxFilePatchBytes: 1000, MaxFiles: 1}}.Compress(ctx)

	if len(got.Chunks) != 1 || got.Chunks[0].File != "a.go" {
		t.Fatalf("Chunks = %#v, want only a.go", got.Chunks)
	}
	if len(got.SkippedFiles) != 1 || got.SkippedFiles[0] != "b.go" {
		t.Fatalf("SkippedFiles = %#v, want b.go", got.SkippedFiles)
	}
}

func TestCompressorTruncatesLargePatchChunk(t *testing.T) {
	content := "File: a.go\nPatch:\n" + strings.Repeat("head", 40) + "\n" + strings.Repeat("tail", 40)
	ctx := ReviewContext{
		Chunks: []Chunk{
			{ID: "a", File: "a.go", Kind: ChunkPatch, Content: content, Tokens: EstimateTokens(content), Score: 100},
		},
	}

	got := Compressor{Budget: TokenBudget{MaxContextTokens: 1000, MaxFilePatchBytes: 120, MaxFiles: 10}}.Compress(ctx)

	if len(got.Chunks) != 1 {
		t.Fatalf("kept chunks = %d, want 1", len(got.Chunks))
	}
	if len([]byte(got.Chunks[0].Content)) > 120 {
		t.Fatalf("truncated content bytes = %d, want <= 120", len([]byte(got.Chunks[0].Content)))
	}
	if !strings.Contains(got.Chunks[0].Content, "patch truncated") {
		t.Fatalf("truncated content missing marker: %q", got.Chunks[0].Content)
	}
}
