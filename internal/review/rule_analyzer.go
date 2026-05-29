package review

import (
	"fmt"
	"strings"

	"aireview/internal/diff"
)

const (
	singleFileLargeChangeThreshold = 300
	totalLargeChangeThreshold      = 800
)

type RuleAnalyzer struct{}

func (a RuleAnalyzer) Analyze(pr PullRequest) []Finding {
	var findings []Finding
	totalChanges := 0
	hasSourceChange := false
	hasTestChange := false

	for _, file := range pr.Files {
		classification := classify(file)
		changeCount := file.Additions + file.Deletions
		totalChanges += changeCount

		if classification.FileKind == diff.FileKindSource {
			hasSourceChange = true
		}
		if classification.FileKind == diff.FileKindTest {
			hasTestChange = true
		}

		if changeCount > singleFileLargeChangeThreshold {
			findings = append(findings, largeFileFinding(file, changeCount))
		}
		switch classification.FileKind {
		case diff.FileKindConfig:
			findings = append(findings, configFinding(file))
		case diff.FileKindDependency:
			findings = append(findings, dependencyFinding(file))
		}

		patch, err := diff.ParsePatch(file.Patch)
		if err != nil || len(patch.AddedLines) == 0 {
			continue
		}
		findings = append(findings, scanAddedLines(file, classification, patch)...)
	}

	if totalChanges > totalLargeChangeThreshold {
		findings = append(findings, largePRFinding(totalChanges))
	}
	if hasSourceChange && !hasTestChange {
		findings = append(findings, missingTestsFinding())
	}

	return findings
}

func classify(file ChangedFile) diff.FileClassification {
	classification := diff.ClassifyFile(file.Path)
	if file.Language != "" {
		classification.Language = file.Language
	}
	if file.FileKind != "" {
		classification.FileKind = file.FileKind
	}
	return classification
}

func largeFileFinding(file ChangedFile, changeCount int) Finding {
	return Finding{
		Severity:   SeverityMedium,
		Confidence: 0.8,
		Category:   CategoryMaintainability,
		File:       file.Path,
		Title:      "Large file change",
		Evidence:   fmt.Sprintf("%s changes %d lines (+%d -%d)", file.Path, changeCount, file.Additions, file.Deletions),
		Suggestion: "Review this file in smaller logical sections and verify tests cover the changed behavior.",
	}
}

func largePRFinding(totalChanges int) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.75,
		Category:        CategoryMaintainability,
		Title:           "Large pull request",
		Evidence:        fmt.Sprintf("This PR changes %d lines across all files", totalChanges),
		Suggestion:      "Consider splitting unrelated changes or increasing reviewer attention on high-risk files.",
		NeedsHumanCheck: true,
	}
}

func configFinding(file ChangedFile) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.75,
		Category:        CategoryMaintainability,
		File:            file.Path,
		Title:           "Configuration file changed",
		Evidence:        fmt.Sprintf("%s is classified as a configuration file", file.Path),
		Suggestion:      "Verify the configuration change against the target environment and deployment defaults.",
		NeedsHumanCheck: true,
	}
}

func dependencyFinding(file ChangedFile) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.8,
		Category:        CategoryCompatibility,
		File:            file.Path,
		Title:           "Dependency file changed",
		Evidence:        fmt.Sprintf("%s is classified as a dependency file", file.Path),
		Suggestion:      "Check lockfile consistency, compatibility impact, and whether dependency changes are intentional.",
		NeedsHumanCheck: true,
	}
}

func missingTestsFinding() Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.7,
		Category:        CategoryTestRisk,
		Title:           "Source changes without test changes",
		Evidence:        "At least one source file changed, but no test file changes were detected.",
		Suggestion:      "Add or update tests for the changed behavior, or document why existing tests are sufficient.",
		NeedsHumanCheck: true,
	}
}

func scanAddedLines(file ChangedFile, classification diff.FileClassification, patch diff.PatchSummary) []Finding {
	var findings []Finding
	for _, line := range patch.AddedLines {
		if keyword, ok := sensitiveKeyword(line.Content); ok {
			findings = append(findings, sensitiveKeywordFinding(file, line.Line, keyword))
		}

		switch classification.Language {
		case "Go":
			if mentionsErr(line.Content) && !diff.HasAddedLineNear(patch.AddedLines, line.Line, 3, handlesGoError) {
				findings = append(findings, goErrorHandlingFinding(file, line.Line))
			}
		case "Python":
			if isBroadPythonException(line.Content) && !diff.HasAddedLineNear(patch.AddedLines, line.Line, 3, handlesException) {
				findings = append(findings, broadExceptionFinding(file, line.Line, "Python broad exception handler"))
			}
		case "Java", "TypeScript", "JavaScript":
			if isCatchLine(line.Content) && !diff.HasAddedLineNear(patch.AddedLines, line.Line, 3, handlesException) {
				findings = append(findings, broadExceptionFinding(file, line.Line, "Empty or unhandled catch block"))
			}
		}
	}
	return findings
}

func sensitiveKeyword(content string) (string, bool) {
	lower := strings.ToLower(content)
	for _, keyword := range []string{"password", "token", "secret", "apikey", "api_key"} {
		if strings.Contains(lower, keyword) {
			return keyword, true
		}
	}
	return "", false
}

func sensitiveKeywordFinding(file ChangedFile, line int, keyword string) Finding {
	return Finding{
		Severity:        SeverityHigh,
		Confidence:      0.85,
		Category:        CategorySecurity,
		File:            file.Path,
		Line:            line,
		Title:           "Sensitive keyword added",
		Evidence:        fmt.Sprintf("Added line contains sensitive keyword %q", keyword),
		Suggestion:      "Verify this is not a hard-coded secret and use environment or secret-management configuration when needed.",
		NeedsHumanCheck: true,
	}
}

func mentionsErr(content string) bool {
	return strings.Contains(content, "err")
}

func handlesGoError(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	return strings.Contains(lower, "return") ||
		strings.Contains(lower, "panic(") ||
		strings.Contains(lower, "log.") ||
		strings.Contains(lower, "fmt.") ||
		strings.Contains(lower, "t.fatal") ||
		strings.Contains(lower, "continue") ||
		strings.Contains(lower, "break")
}

func goErrorHandlingFinding(file ChangedFile, line int) Finding {
	return Finding{
		Severity:        SeverityLow,
		Confidence:      0.6,
		Category:        CategoryCorrectness,
		File:            file.Path,
		Line:            line,
		Title:           "Potentially incomplete error handling",
		Evidence:        "Added Go line mentions err without nearby return, logging, panic, or control-flow handling.",
		Suggestion:      "Verify the error is handled or intentionally ignored.",
		NeedsHumanCheck: true,
	}
}

func isBroadPythonException(content string) bool {
	lower := strings.ToLower(content)
	return strings.Contains(lower, "except exception")
}

func isCatchLine(content string) bool {
	return strings.Contains(strings.ToLower(content), "catch")
}

func handlesException(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" || lower == "{" || lower == "}" || lower == "pass" {
		return false
	}
	return strings.Contains(lower, "throw") ||
		strings.Contains(lower, "raise") ||
		strings.Contains(lower, "return") ||
		strings.Contains(lower, "log") ||
		strings.Contains(lower, "print(")
}

func broadExceptionFinding(file ChangedFile, line int, title string) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.65,
		Category:        CategoryCorrectness,
		File:            file.Path,
		Line:            line,
		Title:           title,
		Evidence:        "Added exception handling does not include nearby logging, rethrow, return, or equivalent handling.",
		Suggestion:      "Handle the exception explicitly, log useful context, or rethrow when the caller should decide.",
		NeedsHumanCheck: true,
	}
}
