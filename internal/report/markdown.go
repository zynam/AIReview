package report

import (
	"fmt"
	"html"
	"io"
	"strings"

	"aireview/internal/github"
	"aireview/internal/review"
)

func WriteMarkdown(w io.Writer, ref github.PRRef, report review.ReviewReport, warning error) error {
	if _, err := fmt.Fprintln(w, "# AI PR Review Report"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "\nPR: `%s/%s#%d`\n", ref.Owner, ref.Repo, ref.Number); err != nil {
		return err
	}
	if warning != nil {
		if _, err := fmt.Fprintf(w, "\n> Warning: %s\n", escapeInline(warning.Error())); err != nil {
			return err
		}
	}

	if err := writeMarkdownTextSection(w, "Summary", report.Summary); err != nil {
		return err
	}
	if err := writeMarkdownListSection(w, "Impact", report.Impact); err != nil {
		return err
	}
	if err := writeMarkdownFindings(w, report.Findings); err != nil {
		return err
	}
	if err := writeMarkdownTextSection(w, "Test Assessment", report.TestAssessment); err != nil {
		return err
	}
	return writeMarkdownListSection(w, "Skipped Files", report.SkippedFiles)
}

func writeMarkdownTextSection(w io.Writer, title string, body string) error {
	if _, err := fmt.Fprintf(w, "\n## %s\n\n", title); err != nil {
		return err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		body = "No content."
	}
	_, err := fmt.Fprintf(w, "%s\n", body)
	return err
}

func writeMarkdownListSection(w io.Writer, title string, items []string) error {
	if _, err := fmt.Fprintf(w, "\n## %s\n\n", title); err != nil {
		return err
	}
	if len(items) == 0 {
		_, err := fmt.Fprintln(w, "None.")
		return err
	}
	for _, item := range items {
		if _, err := fmt.Fprintf(w, "- %s\n", item); err != nil {
			return err
		}
	}
	return nil
}

func writeMarkdownFindings(w io.Writer, findings []review.Finding) error {
	if _, err := fmt.Fprintln(w, "\n## Findings"); err != nil {
		return err
	}
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "\nNone.")
		return err
	}

	for _, finding := range findings {
		if _, err := fmt.Fprintf(w, "\n### [%s] %s\n\n", strings.ToUpper(string(finding.Severity)), finding.Title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "- Location: `%s`\n", findingLocation(finding)); err != nil {
			return err
		}
		if finding.Category != "" {
			if _, err := fmt.Fprintf(w, "- Category: `%s`\n", finding.Category); err != nil {
				return err
			}
		}
		if finding.Confidence > 0 {
			if _, err := fmt.Fprintf(w, "- Confidence: `%.2f`\n", finding.Confidence); err != nil {
				return err
			}
		}
		if finding.Evidence != "" {
			if _, err := fmt.Fprintf(w, "- Evidence: %s\n", finding.Evidence); err != nil {
				return err
			}
		}
		if finding.Suggestion != "" {
			if _, err := fmt.Fprintf(w, "- Suggestion: %s\n", finding.Suggestion); err != nil {
				return err
			}
		}
		if finding.NeedsHumanCheck {
			if _, err := fmt.Fprintln(w, "- Needs human check: `true`"); err != nil {
				return err
			}
		}
	}
	return nil
}

func escapeInline(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}
