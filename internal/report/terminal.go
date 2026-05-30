package report

import (
	"fmt"
	"io"
	"strings"

	"aireview/internal/github"
	"aireview/internal/review"
)

func WriteTerminal(w io.Writer, ref github.PRRef, report review.ReviewReport, warning error) error {
	if _, err := fmt.Fprintf(w, "AI PR Review %s/%s#%d\n\n", ref.Owner, ref.Repo, ref.Number); err != nil {
		return err
	}
	if warning != nil {
		if _, err := fmt.Fprintf(w, "Warning: %v\n\n", warning); err != nil {
			return err
		}
	}

	if err := writeSection(w, "Summary", report.Summary); err != nil {
		return err
	}
	if len(report.Impact) > 0 {
		if _, err := fmt.Fprintln(w, "\nImpact"); err != nil {
			return err
		}
		for _, item := range report.Impact {
			if _, err := fmt.Fprintf(w, "- %s\n", item); err != nil {
				return err
			}
		}
	}

	if _, err := fmt.Fprintf(w, "\nFindings: %d\n", len(report.Findings)); err != nil {
		return err
	}
	for _, finding := range report.Findings {
		if err := writeTerminalFinding(w, finding); err != nil {
			return err
		}
	}

	if strings.TrimSpace(report.TestAssessment) != "" {
		if err := writeSection(w, "\nTest Assessment", report.TestAssessment); err != nil {
			return err
		}
	}
	if len(report.SkippedFiles) > 0 {
		if _, err := fmt.Fprintln(w, "\nSkipped Files"); err != nil {
			return err
		}
		for _, file := range report.SkippedFiles {
			if _, err := fmt.Fprintf(w, "- %s\n", file); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeSection(w io.Writer, title string, body string) error {
	if _, err := fmt.Fprintln(w, title); err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		body = "No content."
	}
	_, err := fmt.Fprintln(w, body)
	return err
}

func writeTerminalFinding(w io.Writer, finding review.Finding) error {
	location := findingLocation(finding)
	if _, err := fmt.Fprintf(w, "[%s] %s %s\n", strings.ToUpper(string(finding.Severity)), location, finding.Title); err != nil {
		return err
	}
	if finding.Evidence != "" {
		if _, err := fmt.Fprintf(w, "Evidence: %s\n", finding.Evidence); err != nil {
			return err
		}
	}
	if finding.Suggestion != "" {
		if _, err := fmt.Fprintf(w, "Suggestion: %s\n", finding.Suggestion); err != nil {
			return err
		}
	}
	return nil
}

func findingLocation(finding review.Finding) string {
	if finding.File == "" {
		return "PR"
	}
	if finding.Line > 0 {
		return fmt.Sprintf("%s:%d", finding.File, finding.Line)
	}
	return finding.File
}
