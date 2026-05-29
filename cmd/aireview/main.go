package main

import (
	"fmt"
	"os"

	"aireview/internal/config"
	"aireview/internal/diff"
	"aireview/internal/github"
	"aireview/internal/llm"
	"aireview/internal/review"

	"github.com/spf13/cobra"
)

type reviewOptions struct {
	owner         string
	repo          string
	prNumber      int
	output        string
	format        string
	configPath    string
	minSeverity   string
	minConfidence float64
	maxFiles      int
}

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aireview",
		Short: "AI-assisted GitHub pull request reviewer",
		Long:  "AIReview analyzes GitHub pull request changes and generates a local review report.",
	}

	cmd.AddCommand(newReviewCommand())
	return cmd
}

func newReviewCommand() *cobra.Command {
	var opts reviewOptions

	cmd := &cobra.Command{
		Use:   "review <pr-url>",
		Short: "Analyze a GitHub pull request",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				_, err := github.ParsePRURL(args[0])
				return err
			}
			if opts.owner != "" && opts.repo != "" && opts.prNumber > 0 {
				return nil
			}
			return fmt.Errorf("provide either <pr-url> or --owner, --repo and --pr")
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := resolvePRRef(args, opts)
			if err != nil {
				return err
			}
			client := github.NewClient()
			pr, err := client.GetPullRequest(cmd.Context(), ref)
			if err != nil {
				return err
			}
			cfg, err := config.Load(opts.configPath)
			if err != nil {
				return err
			}
			classifyFiles(pr.Files)
			findings := review.RuleAnalyzer{}.Analyze(pr)
			report, llmErr := llm.NewOpenAICompatibleProvider(cfg).Review(cmd.Context(), llm.ReviewRequest{
				PullRequest:  pr,
				RuleFindings: findings,
				Config:       cfg,
			})
			printPRSummary(pr, findings, report.Report, llmErr)
			if llmErr != nil {
				return llmErr
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.owner, "owner", "", "GitHub repository owner")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "GitHub repository name")
	cmd.Flags().IntVar(&opts.prNumber, "pr", 0, "GitHub pull request number")
	cmd.Flags().StringVar(&opts.output, "output", "", "write Markdown report to this file")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text or markdown")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .aireview.toml")
	cmd.Flags().StringVar(&opts.minSeverity, "min-severity", "", "minimum severity to include: low, medium or high")
	cmd.Flags().Float64Var(&opts.minConfidence, "min-confidence", 0, "minimum confidence to include")
	cmd.Flags().IntVar(&opts.maxFiles, "max-files", 0, "maximum number of changed files to analyze")

	return cmd
}

func resolvePRRef(args []string, opts reviewOptions) (github.PRRef, error) {
	if len(args) == 1 {
		return github.ParsePRURL(args[0])
	}
	return github.PRRef{
		Owner:  opts.owner,
		Repo:   opts.repo,
		Number: opts.prNumber,
	}, nil
}

func classifyFiles(files []review.ChangedFile) {
	for i := range files {
		classification := diff.ClassifyFile(files[i].Path)
		files[i].Language = classification.Language
		files[i].FileKind = classification.FileKind
	}
}

func printPRSummary(pr review.PullRequest, findings []review.Finding, aiReport review.ReviewReport, llmErr error) {
	fmt.Printf("PR #%d %s/%s\n", pr.Number, pr.Owner, pr.Repo)
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("Author: %s\n", pr.Author)
	fmt.Printf("Base: %s\n", pr.BaseSHA)
	fmt.Printf("Head: %s\n", pr.HeadSHA)
	fmt.Printf("Changed files: %d\n", len(pr.Files))
	for _, file := range pr.Files {
		fmt.Printf("- %s (%s, %s/%s, +%d -%d)\n", file.Path, file.Status, emptyAsUnknown(file.Language), emptyAsUnknown(file.FileKind), file.Additions, file.Deletions)
	}
	fmt.Printf("Commits: %d\n", len(pr.Commits))
	for _, commit := range pr.Commits {
		fmt.Printf("- %s %s (%s)\n", shortSHA(commit.SHA), firstLine(commit.Message), commit.Author)
	}
	fmt.Printf("Rule findings: %d\n", len(findings))
	for _, finding := range findings {
		location := finding.File
		if finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", finding.File, finding.Line)
		}
		if location == "" {
			location = "PR"
		}
		fmt.Printf("- [%s] %s %s\n", finding.Severity, location, finding.Title)
	}
	if llmErr != nil {
		fmt.Printf("AI review: failed: %v\n", llmErr)
		return
	}
	fmt.Println("AI review:")
	fmt.Printf("Summary: %s\n", aiReport.Summary)
	fmt.Printf("Findings: %d\n", len(aiReport.Findings))
	for _, finding := range aiReport.Findings {
		location := finding.File
		if finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", finding.File, finding.Line)
		}
		if location == "" {
			location = "PR"
		}
		fmt.Printf("- [%s] %s %s\n", finding.Severity, location, finding.Title)
	}
}

func emptyAsUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func firstLine(message string) string {
	for i, r := range message {
		if r == '\n' || r == '\r' {
			return message[:i]
		}
	}
	return message
}
