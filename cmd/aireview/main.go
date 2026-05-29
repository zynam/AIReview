package main

import (
	"fmt"
	"os"

	"aireview/internal/app"
	"aireview/internal/config"
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
			cfg, err := config.Load(opts.configPath)
			if err != nil {
				return err
			}
			service := app.ReviewService{
				GitHub: github.NewClient(),
				LLM:    llm.NewOpenAICompatibleProvider(cfg),
			}
			report, err := service.ReviewPR(cmd.Context(), app.ReviewPRRequest{
				Ref:           ref,
				Config:        cfg,
				MinSeverity:   opts.minSeverity,
				MinConfidence: opts.minConfidence,
				MaxFiles:      opts.maxFiles,
			})
			if report.Summary != "" || len(report.Findings) > 0 || len(report.SkippedFiles) > 0 {
				printReviewReport(ref, report, err)
			}
			return err
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

func printReviewReport(ref github.PRRef, report review.ReviewReport, err error) {
	fmt.Printf("AI PR Review %s/%s#%d\n", ref.Owner, ref.Repo, ref.Number)
	fmt.Println()
	if err != nil {
		fmt.Printf("Warning: %v\n\n", err)
	}
	fmt.Println("Summary")
	fmt.Println(report.Summary)
	if len(report.Impact) > 0 {
		fmt.Println()
		fmt.Println("Impact")
		for _, item := range report.Impact {
			fmt.Printf("- %s\n", item)
		}
	}
	fmt.Println()
	fmt.Printf("Findings: %d\n", len(report.Findings))
	for _, finding := range report.Findings {
		location := finding.File
		if finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", finding.File, finding.Line)
		}
		if location == "" {
			location = "PR"
		}
		fmt.Printf("- [%s] %s %s\n", finding.Severity, location, finding.Title)
		if finding.Evidence != "" {
			fmt.Printf("  Evidence: %s\n", finding.Evidence)
		}
		if finding.Suggestion != "" {
			fmt.Printf("  Suggestion: %s\n", finding.Suggestion)
		}
	}
	if report.TestAssessment != "" {
		fmt.Println()
		fmt.Println("Test Assessment")
		fmt.Println(report.TestAssessment)
	}
	if len(report.SkippedFiles) > 0 {
		fmt.Println()
		fmt.Println("Skipped Files")
		for _, file := range report.SkippedFiles {
			fmt.Printf("- %s\n", file)
		}
	}
}
