package agent

import (
	"testing"

	"aireview/internal/review"
)

func TestValidatorNormalizesConfidenceAndSeverity(t *testing.T) {
	got := Validator{}.ValidateFinding(nil, review.Finding{
		Severity:   "critical",
		Confidence: 1.5,
	})

	if got.Severity != review.SeverityLow {
		t.Fatalf("Severity = %q, want low", got.Severity)
	}
	if got.Confidence != 1 {
		t.Fatalf("Confidence = %.2f, want 1", got.Confidence)
	}
}

func TestValidatorMarksUnknownFileForHumanCheck(t *testing.T) {
	pr := review.PullRequest{
		Files: []review.ChangedFile{{Path: "internal/service.go", Patch: "@@ -10,0 +10,2 @@\n+func run() {}\n+return nil"}},
	}
	report := review.ReviewReport{
		Findings: []review.Finding{{
			Severity:   review.SeverityHigh,
			Confidence: 0.8,
			File:       "internal/missing.go",
			Line:       10,
			Title:      "missing",
		}},
	}

	got := Validator{}.ValidateReport(pr, report)
	finding := got.Findings[0]

	if finding.Severity != review.SeverityLow {
		t.Fatalf("Severity = %q, want low", finding.Severity)
	}
	if !finding.NeedsHumanCheck {
		t.Fatal("NeedsHumanCheck = false, want true")
	}
}

func TestValidatorMarksLineOutsidePatchForHumanCheck(t *testing.T) {
	pr := review.PullRequest{
		Files: []review.ChangedFile{{Path: "internal/service.go", Patch: "@@ -10,0 +10,2 @@\n+func run() {}\n+return nil"}},
	}
	report := review.ReviewReport{
		Findings: []review.Finding{{
			Severity:   review.SeverityHigh,
			Confidence: 0.8,
			File:       "internal/service.go",
			Line:       99,
			Title:      "bad line",
		}},
	}

	got := Validator{}.ValidateReport(pr, report)
	finding := got.Findings[0]

	if finding.Severity != review.SeverityLow {
		t.Fatalf("Severity = %q, want low", finding.Severity)
	}
	if !finding.NeedsHumanCheck {
		t.Fatal("NeedsHumanCheck = false, want true")
	}
}

func TestValidatorKeepsFindingOnAddedLine(t *testing.T) {
	pr := review.PullRequest{
		Files: []review.ChangedFile{{Path: "internal/service.go", Patch: "@@ -10,0 +10,2 @@\n+func run() {}\n+return nil"}},
	}

	got := Validator{}.ValidateReport(pr, review.ReviewReport{
		Findings: []review.Finding{{
			Severity:   review.SeverityHigh,
			Confidence: -0.2,
			File:       "internal/service.go",
			Line:       11,
			Title:      "valid line",
		}},
	})
	finding := got.Findings[0]

	if finding.Severity != review.SeverityHigh {
		t.Fatalf("Severity = %q, want high", finding.Severity)
	}
	if finding.NeedsHumanCheck {
		t.Fatal("NeedsHumanCheck = true, want false")
	}
	if finding.Confidence != 0 {
		t.Fatalf("Confidence = %.2f, want 0", finding.Confidence)
	}
}

func TestValidatorTreatsEmptyFileAsPRLevelFinding(t *testing.T) {
	got := Validator{}.ValidateFinding(nil, review.Finding{
		Severity:   review.SeverityMedium,
		Confidence: 0.7,
		File:       " ",
		Line:       100,
	})

	if got.File != "" {
		t.Fatalf("File = %q, want empty", got.File)
	}
	if got.Severity != review.SeverityMedium {
		t.Fatalf("Severity = %q, want medium", got.Severity)
	}
	if got.NeedsHumanCheck {
		t.Fatal("NeedsHumanCheck = true, want false")
	}
}
