package reviewcontext

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"aireview/internal/config"
	"aireview/internal/diff"
	"aireview/internal/review"
)

const (
	defaultPRMetaScore      = 80
	defaultCommitScore      = 50
	defaultPatchScore       = 60
	defaultRuleFindingScore = 90
	generatedFileScore      = 20
)

type Builder struct {
	Config config.Config
}

func (b Builder) Build(pr review.PullRequest, ruleFindings []review.Finding) ReviewContext {
	ctx := ReviewContext{
		PullRequest:  pr,
		RuleFindings: append([]review.Finding(nil), ruleFindings...),
	}

	ctx.Chunks = append(ctx.Chunks, b.prMetaChunk(pr))
	for i, commit := range pr.Commits {
		ctx.Chunks = append(ctx.Chunks, b.commitChunk(i, commit))
	}
	for i, file := range pr.Files {
		ctx.Chunks = append(ctx.Chunks, b.patchChunk(i, file))
	}
	for i, finding := range ruleFindings {
		ctx.Chunks = append(ctx.Chunks, b.ruleFindingChunk(i, finding))
	}

	return ctx
}

func (b Builder) prMetaChunk(pr review.PullRequest) Chunk {
	content := strings.TrimSpace(fmt.Sprintf(
		"Repository: %s/%s\nPR: #%d\nTitle: %s\nAuthor: %s\nBase SHA: %s\nHead SHA: %s\nBody:\n%s",
		pr.Owner,
		pr.Repo,
		pr.Number,
		pr.Title,
		pr.Author,
		pr.BaseSHA,
		pr.HeadSHA,
		pr.Body,
	))
	return newChunk("pr-meta", "", ChunkPRMeta, content, defaultPRMetaScore)
}

func (b Builder) commitChunk(index int, commit review.Commit) Chunk {
	content := strings.TrimSpace(fmt.Sprintf(
		"Commit: %s\nAuthor: %s\nMessage:\n%s",
		commit.SHA,
		commit.Author,
		commit.Message,
	))
	return newChunk(fmt.Sprintf("commit-%d", index+1), "", ChunkCommit, content, defaultCommitScore)
}

func (b Builder) patchChunk(index int, file review.ChangedFile) Chunk {
	content := strings.TrimSpace(fmt.Sprintf(
		"File: %s\nStatus: %s\nLanguage: %s\nKind: %s\nAdditions: %d\nDeletions: %d\nRisk hints: %s\nPatch:\n%s",
		file.Path,
		file.Status,
		file.Language,
		fileKind(file),
		file.Additions,
		file.Deletions,
		mustJSON(file.RiskHints),
		file.Patch,
	))
	return newChunk(fmt.Sprintf("patch-%d-%s", index+1, stableIDPart(file.Path)), file.Path, ChunkPatch, content, b.fileScore(file))
}

func (b Builder) ruleFindingChunk(index int, finding review.Finding) Chunk {
	content := strings.TrimSpace(fmt.Sprintf(
		"Rule finding: %s\nSeverity: %s\nConfidence: %.2f\nCategory: %s\nFile: %s\nLine: %d\nEvidence: %s\nSuggestion: %s\nNeeds human check: %t",
		finding.Title,
		finding.Severity,
		finding.Confidence,
		finding.Category,
		finding.File,
		finding.Line,
		finding.Evidence,
		finding.Suggestion,
		finding.NeedsHumanCheck,
	))
	score := float64(defaultRuleFindingScore)
	if finding.Severity == review.SeverityHigh {
		score += 30
	}
	return newChunk(fmt.Sprintf("rule-finding-%d", index+1), finding.File, ChunkRuleFinding, content, score)
}

func (b Builder) fileScore(file review.ChangedFile) float64 {
	kind := fileKind(file)
	if kind == diff.FileKindGenerated {
		return generatedFileScore
	}

	score := float64(defaultPatchScore)
	switch kind {
	case diff.FileKindSource:
		score += 25
	case diff.FileKindTest:
		score += 15
	case diff.FileKindConfig, diff.FileKindDependency:
		score += 20
	case diff.FileKindDocumentation:
		score -= 10
	}
	if file.Additions+file.Deletions > 300 {
		score += 10
	}
	if len(file.RiskHints) > 0 {
		score += 15
	}
	if pathMatchesAny(file.Path, b.Config.HighRiskPaths) {
		score += 30
	}
	return score
}

func fileKind(file review.ChangedFile) string {
	if strings.TrimSpace(file.FileKind) != "" {
		return file.FileKind
	}
	return diff.ClassifyFile(file.Path).FileKind
}

func newChunk(id string, file string, kind ChunkKind, content string, score float64) Chunk {
	return Chunk{
		ID:      id,
		File:    file,
		Kind:    kind,
		Content: content,
		Tokens:  EstimateTokens(content),
		Score:   score,
	}
}

func EstimateTokens(content string) int {
	if content == "" {
		return 0
	}
	return (len([]byte(content)) + 3) / 4
}

func stableIDPart(path string) string {
	id := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	id = strings.Trim(id, "/")
	id = strings.NewReplacer("/", "-", " ", "-", ".", "-").Replace(id)
	if id == "" {
		return "file"
	}
	return id
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
		if strings.Contains(normalized, strings.Trim(pattern, "*")) {
			return true
		}
	}
	return false
}

func mustJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(body)
}
