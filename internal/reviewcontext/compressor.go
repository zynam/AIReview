package reviewcontext

import (
	"sort"
	"strings"

	"aireview/internal/review"
)

type Compressor struct {
	Budget TokenBudget
}

func (c Compressor) Compress(ctx ReviewContext) ReviewContext {
	budget := c.Budget.normalize()
	result := ReviewContext{
		PullRequest:  ctx.PullRequest,
		RuleFindings: append([]review.Finding(nil), ctx.RuleFindings...),
		Chunks:       make([]Chunk, 0, len(ctx.Chunks)),
		SkippedFiles: append([]string(nil), ctx.SkippedFiles...),
	}

	chunks := append([]Chunk(nil), ctx.Chunks...)
	sort.SliceStable(chunks, func(i, j int) bool {
		if chunks[i].Score != chunks[j].Score {
			return chunks[i].Score > chunks[j].Score
		}
		return chunks[i].ID < chunks[j].ID
	})

	keptPatchFiles := make(map[string]struct{})
	usedTokens := 0
	for _, chunk := range chunks {
		chunk = c.fitChunk(chunk, budget)
		if chunk.Kind == ChunkPatch && chunk.File != "" {
			if _, exists := keptPatchFiles[chunk.File]; !exists && len(keptPatchFiles) >= budget.MaxFiles {
				result.SkippedFiles = appendSkippedFile(result.SkippedFiles, chunk.File)
				continue
			}
		}
		if budget.MaxContextTokens > 0 && usedTokens+chunk.Tokens > budget.MaxContextTokens {
			result.SkippedFiles = appendSkippedFile(result.SkippedFiles, chunk.File)
			continue
		}
		result.Chunks = append(result.Chunks, chunk)
		usedTokens += chunk.Tokens
		if chunk.Kind == ChunkPatch && chunk.File != "" {
			keptPatchFiles[chunk.File] = struct{}{}
		}
	}

	return result
}

func (c Compressor) fitChunk(chunk Chunk, budget TokenBudget) Chunk {
	if chunk.Kind != ChunkPatch || budget.MaxFilePatchBytes <= 0 {
		return chunk
	}
	if len([]byte(chunk.Content)) <= budget.MaxFilePatchBytes {
		return chunk
	}
	chunk.Content = truncateMiddle(chunk.Content, budget.MaxFilePatchBytes)
	chunk.Tokens = EstimateTokens(chunk.Content)
	return chunk
}

func truncateMiddle(content string, maxBytes int) string {
	if maxBytes <= 0 || len([]byte(content)) <= maxBytes {
		return content
	}

	marker := "\n\n... patch truncated by token budget ...\n\n"
	markerBytes := len([]byte(marker))
	if maxBytes <= markerBytes {
		return string([]byte(content)[:maxBytes])
	}

	keepBytes := maxBytes - markerBytes
	headBytes := keepBytes / 2
	tailBytes := keepBytes - headBytes
	body := []byte(content)
	return strings.TrimSpace(string(body[:headBytes])) + marker + strings.TrimSpace(string(body[len(body)-tailBytes:]))
}

func appendSkippedFile(files []string, file string) []string {
	file = strings.TrimSpace(file)
	if file == "" {
		return files
	}
	for _, existing := range files {
		if existing == file {
			return files
		}
	}
	return append(files, file)
}
