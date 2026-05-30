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
		Title:      "单文件变更较大",
		Evidence:   fmt.Sprintf("%s 变更 %d 行（+%d -%d）", file.Path, changeCount, file.Additions, file.Deletions),
		Suggestion: "按逻辑分块审查该文件，并确认测试覆盖了变更行为。",
	}
}

func largePRFinding(totalChanges int) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.75,
		Category:        CategoryMaintainability,
		Title:           "PR 变更规模较大",
		Evidence:        fmt.Sprintf("该 PR 所有文件合计变更 %d 行", totalChanges),
		Suggestion:      "考虑拆分无关变更，或对高风险文件投入更多审查关注。",
		NeedsHumanCheck: true,
	}
}

func configFinding(file ChangedFile) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.75,
		Category:        CategoryMaintainability,
		File:            file.Path,
		Title:           "配置文件发生变更",
		Evidence:        fmt.Sprintf("%s 被识别为配置文件", file.Path),
		Suggestion:      "确认该配置变更与目标环境和部署默认值兼容。",
		NeedsHumanCheck: true,
	}
}

func dependencyFinding(file ChangedFile) Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.8,
		Category:        CategoryCompatibility,
		File:            file.Path,
		Title:           "依赖文件发生变更",
		Evidence:        fmt.Sprintf("%s 被识别为依赖文件", file.Path),
		Suggestion:      "检查锁文件一致性、兼容性影响，并确认依赖变更是否符合预期。",
		NeedsHumanCheck: true,
	}
}

func missingTestsFinding() Finding {
	return Finding{
		Severity:        SeverityMedium,
		Confidence:      0.7,
		Category:        CategoryTestRisk,
		Title:           "源码变更缺少对应测试变更",
		Evidence:        "检测到至少一个源码文件发生变更，但没有检测到测试文件变更。",
		Suggestion:      "为变更行为新增或更新测试，或说明现有测试为何已经足够。",
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
				findings = append(findings, broadExceptionFinding(file, line.Line, "Python 宽泛异常处理可能缺少处理逻辑"))
			}
		case "Java", "TypeScript", "JavaScript":
			if isCatchLine(line.Content) && !diff.HasAddedLineNear(patch.AddedLines, line.Line, 3, handlesException) {
				findings = append(findings, broadExceptionFinding(file, line.Line, "catch 块可能缺少处理逻辑"))
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
		Title:           "新增内容包含敏感关键词",
		Evidence:        fmt.Sprintf("新增行包含敏感关键词 %q", keyword),
		Suggestion:      "确认这不是硬编码密钥；如需配置敏感值，应使用环境变量或密钥管理方案。",
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
		Title:           "错误处理可能不完整",
		Evidence:        "新增 Go 代码提到 err，但附近没有 return、日志、panic 或控制流处理。",
		Suggestion:      "确认该错误已被处理，或明确说明为什么可以忽略。",
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
		Evidence:        "新增异常处理附近没有日志、重新抛出、return 或等价处理逻辑。",
		Suggestion:      "显式处理异常、记录有用上下文，或在应由调用方决策时重新抛出。",
		NeedsHumanCheck: true,
	}
}
