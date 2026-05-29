package review

import (
	"strings"
	"testing"
)

func TestRuleAnalyzerDetectsFileLevelRisks(t *testing.T) {
	pr := PullRequest{
		Files: []ChangedFile{
			{
				Path:      "internal/auth/check.go",
				Additions: 20,
				Deletions: 4,
				Patch: `@@ -1,2 +1,4 @@
 package auth
+const apiToken = "dev-token"`,
			},
			{
				Path:      ".github/workflows/ci.yml",
				Additions: 3,
				Deletions: 1,
			},
			{
				Path:      "go.mod",
				Additions: 1,
			},
		},
	}

	findings := RuleAnalyzer{}.Analyze(pr)
	assertFinding(t, findings, CategorySecurity, "Sensitive keyword added")
	assertFinding(t, findings, CategoryMaintainability, "Configuration file changed")
	assertFinding(t, findings, CategoryCompatibility, "Dependency file changed")
	assertFinding(t, findings, CategoryTestRisk, "Source changes without test changes")
}

func TestRuleAnalyzerDoesNotReportMissingTestsWhenTestsChanged(t *testing.T) {
	pr := PullRequest{
		Files: []ChangedFile{
			{Path: "internal/auth/check.go", Additions: 5},
			{Path: "internal/auth/check_test.go", Additions: 10},
		},
	}

	findings := RuleAnalyzer{}.Analyze(pr)
	if hasFinding(findings, CategoryTestRisk, "Source changes without test changes") {
		t.Fatal("Analyze() reported missing tests even though a test file changed")
	}
}

func TestRuleAnalyzerDetectsPatchLineRisks(t *testing.T) {
	pr := PullRequest{
		Files: []ChangedFile{
			{
				Path: "internal/service.go",
				Patch: `@@ -10,2 +10,4 @@
 func run() {
+	if err := do(); err != nil {
+	}
 }`,
			},
			{
				Path: "worker.py",
				Patch: `@@ -20,2 +20,4 @@
 def run():
+    except Exception:
+        pass`,
			},
			{
				Path: "web/app.ts",
				Patch: `@@ -30,2 +30,4 @@
 try {
+} catch (err) {
+}`,
			},
		},
	}

	findings := RuleAnalyzer{}.Analyze(pr)
	assertFinding(t, findings, CategoryCorrectness, "Potentially incomplete error handling")
	assertFinding(t, findings, CategoryCorrectness, "Python broad exception handler")
	assertFinding(t, findings, CategoryCorrectness, "Empty or unhandled catch block")
}

func TestRuleAnalyzerSuppressesHandledPatchLineRisks(t *testing.T) {
	pr := PullRequest{
		Files: []ChangedFile{
			{
				Path: "internal/service.go",
				Patch: `@@ -10,2 +10,5 @@
 func run() error {
+	if err := do(); err != nil {
+		return err
+	}
 }`,
			},
			{
				Path: "worker.py",
				Patch: `@@ -20,2 +20,4 @@
 def run():
+    except Exception:
+        raise`,
			},
		},
	}

	findings := RuleAnalyzer{}.Analyze(pr)
	if hasFinding(findings, CategoryCorrectness, "Potentially incomplete error handling") {
		t.Fatal("Analyze() reported handled Go error as incomplete")
	}
	if hasFinding(findings, CategoryCorrectness, "Python broad exception handler") {
		t.Fatal("Analyze() reported re-raised Python exception as unhandled")
	}
}

func TestRuleAnalyzerDetectsLargeChangeRisks(t *testing.T) {
	pr := PullRequest{
		Files: []ChangedFile{
			{Path: "internal/large.go", Additions: 301},
			{Path: "internal/another.go", Additions: 500},
		},
	}

	findings := RuleAnalyzer{}.Analyze(pr)
	assertFinding(t, findings, CategoryMaintainability, "Large file change")
	assertFinding(t, findings, CategoryMaintainability, "Large pull request")
}

func assertFinding(t *testing.T, findings []Finding, category Category, title string) {
	t.Helper()
	if !hasFinding(findings, category, title) {
		t.Fatalf("missing finding category=%q title containing %q; got %#v", category, title, findings)
	}
}

func hasFinding(findings []Finding, category Category, title string) bool {
	for _, finding := range findings {
		if finding.Category == category && strings.Contains(finding.Title, title) {
			return true
		}
	}
	return false
}
