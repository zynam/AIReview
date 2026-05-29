package review

import (
	"sort"
	"strconv"
	"strings"
)

func MergeReports(aiReport ReviewReport, ruleFindings []Finding) ReviewReport {
	merged := aiReport
	merged.Findings = MergeFindings(ruleFindings, aiReport.Findings)
	return merged
}

func MergeFindings(ruleFindings []Finding, aiFindings []Finding) []Finding {
	byKey := make(map[string]Finding)
	order := make([]string, 0, len(ruleFindings)+len(aiFindings))

	add := func(finding Finding) {
		key := findingKey(finding)
		if existing, ok := byKey[key]; ok {
			byKey[key] = mergeFinding(existing, finding)
			return
		}
		byKey[key] = finding
		order = append(order, key)
	}

	for _, finding := range ruleFindings {
		add(finding)
	}
	for _, finding := range aiFindings {
		add(finding)
	}

	findings := make([]Finding, 0, len(order))
	for _, key := range order {
		findings = append(findings, byKey[key])
	}
	SortFindings(findings)
	return findings
}

func FilterFindings(findings []Finding, minSeverity Severity, minConfidence float64) []Finding {
	if minSeverity == "" && minConfidence <= 0 {
		return findings
	}

	filtered := make([]Finding, 0, len(findings))
	for _, finding := range findings {
		if minSeverity != "" && compareSeverity(finding.Severity, minSeverity) < 0 {
			continue
		}
		if minConfidence > 0 && finding.Confidence < minConfidence {
			continue
		}
		filtered = append(filtered, finding)
	}
	return filtered
}

func SortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		left := findings[i]
		right := findings[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if left.Confidence != right.Confidence {
			return left.Confidence > right.Confidence
		}
		if left.File != right.File {
			return left.File < right.File
		}
		if left.Line != right.Line {
			return left.Line < right.Line
		}
		return left.Title < right.Title
	})
}

func ParseSeverity(value string) (Severity, bool) {
	switch Severity(strings.ToLower(strings.TrimSpace(value))) {
	case "":
		return "", true
	case SeverityLow:
		return SeverityLow, true
	case SeverityMedium:
		return SeverityMedium, true
	case SeverityHigh:
		return SeverityHigh, true
	default:
		return "", false
	}
}

func mergeFinding(left Finding, right Finding) Finding {
	merged := left
	if compareSeverity(right.Severity, merged.Severity) > 0 {
		merged.Severity = right.Severity
	}
	if right.Confidence > merged.Confidence {
		merged.Confidence = right.Confidence
	}
	if len(strings.TrimSpace(right.Suggestion)) > len(strings.TrimSpace(merged.Suggestion)) {
		merged.Suggestion = right.Suggestion
	}
	if len(strings.TrimSpace(right.Evidence)) > len(strings.TrimSpace(merged.Evidence)) {
		merged.Evidence = right.Evidence
	}
	if merged.Category == "" {
		merged.Category = right.Category
	}
	if merged.File == "" {
		merged.File = right.File
	}
	if merged.Line == 0 {
		merged.Line = right.Line
	}
	if merged.Title == "" {
		merged.Title = right.Title
	}
	merged.NeedsHumanCheck = merged.NeedsHumanCheck || right.NeedsHumanCheck
	return merged
}

func findingKey(finding Finding) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(finding.File)),
		intKey(finding.Line),
		strings.ToLower(strings.TrimSpace(finding.Title)),
	}, "\x00")
}

func intKey(value int) string {
	return strconv.Itoa(value)
}

func compareSeverity(left Severity, right Severity) int {
	return severityRank(left) - severityRank(right)
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}
