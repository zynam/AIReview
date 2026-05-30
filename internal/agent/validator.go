package agent

import (
	"strings"

	"aireview/internal/diff"
	"aireview/internal/review"
)

const addedLineRadius = 3

type Validator struct{}

func (v Validator) ValidateReport(pr review.PullRequest, report review.ReviewReport) review.ReviewReport {
	report.Findings = v.ValidateFindings(pr, report.Findings)
	return report
}

func (v Validator) ValidateFindings(pr review.PullRequest, findings []review.Finding) []review.Finding {
	files := changedFileIndex(pr.Files)
	validated := make([]review.Finding, 0, len(findings))
	for _, finding := range findings {
		validated = append(validated, v.ValidateFinding(files, finding))
	}
	return validated
}

func (v Validator) ValidateFinding(files map[string]review.ChangedFile, finding review.Finding) review.Finding {
	finding.Confidence = normalizeConfidence(finding.Confidence)
	if severity, ok := review.ParseSeverity(string(finding.Severity)); !ok || severity == "" {
		finding.Severity = review.SeverityLow
	}

	file := strings.TrimSpace(finding.File)
	if file == "" {
		finding.File = ""
		return finding
	}
	finding.File = file

	changedFile, ok := files[normalizeValidationPath(file)]
	if !ok {
		return markNeedsHumanCheck(finding)
	}
	if finding.Line <= 0 {
		return finding
	}
	if !lineExistsNearPatch(changedFile.Patch, finding.Line) {
		return markNeedsHumanCheck(finding)
	}
	return finding
}

func changedFileIndex(files []review.ChangedFile) map[string]review.ChangedFile {
	index := make(map[string]review.ChangedFile, len(files))
	for _, file := range files {
		index[normalizeValidationPath(file.Path)] = file
	}
	return index
}

func normalizeValidationPath(path string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(path), "\\", "/"))
}

func normalizeConfidence(confidence float64) float64 {
	switch {
	case confidence < 0:
		return 0
	case confidence > 1:
		return 1
	default:
		return confidence
	}
}

func markNeedsHumanCheck(finding review.Finding) review.Finding {
	finding.Severity = review.SeverityLow
	finding.NeedsHumanCheck = true
	return finding
}

func lineExistsNearPatch(patch string, line int) bool {
	summary, err := diff.ParsePatch(patch)
	if err != nil {
		return false
	}
	if len(diff.AddedLinesNear(summary.AddedLines, line, addedLineRadius)) > 0 {
		return true
	}
	return false
}
