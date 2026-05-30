package reviewcontext

import (
	"sort"
	"strings"

	"aireview/internal/diff"
	"aireview/internal/review"
)

type Retriever struct{}

func (r Retriever) Retrieve(ctx ReviewContext) ReviewContext {
	result := ctx
	result.Chunks = append([]Chunk(nil), ctx.Chunks...)

	highSeverityFiles := highSeverityRuleFiles(ctx.RuleFindings)
	sort.SliceStable(result.Chunks, func(i, j int) bool {
		left := retrieveRank(result.Chunks[i], highSeverityFiles)
		right := retrieveRank(result.Chunks[j], highSeverityFiles)
		if left != right {
			return left > right
		}
		if result.Chunks[i].Score != result.Chunks[j].Score {
			return result.Chunks[i].Score > result.Chunks[j].Score
		}
		return result.Chunks[i].ID < result.Chunks[j].ID
	})

	return result
}

func highSeverityRuleFiles(findings []review.Finding) map[string]struct{} {
	files := make(map[string]struct{})
	for _, finding := range findings {
		file := strings.TrimSpace(finding.File)
		if finding.Severity == review.SeverityHigh && file != "" {
			files[normalizeFile(file)] = struct{}{}
		}
	}
	return files
}

func retrieveRank(chunk Chunk, highSeverityFiles map[string]struct{}) int {
	if chunk.Kind == ChunkPRMeta {
		return 1000
	}
	if chunk.Kind == ChunkRuleFinding {
		if _, ok := highSeverityFiles[normalizeFile(chunk.File)]; ok {
			return 950
		}
		return 850
	}
	if _, ok := highSeverityFiles[normalizeFile(chunk.File)]; ok {
		return 900
	}

	switch fileKindFromPath(chunk.File) {
	case diff.FileKindSource:
		return 800
	case diff.FileKindTest:
		return 700
	case diff.FileKindConfig, diff.FileKindDependency:
		return 600
	case diff.FileKindDocumentation:
		return 500
	case diff.FileKindGenerated:
		return 100
	default:
		return 300
	}
}

func fileKindFromPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return diff.ClassifyFile(path).FileKind
}

func normalizeFile(path string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"))
}
