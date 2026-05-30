package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"aireview/internal/app"
	"aireview/internal/config"
	"aireview/internal/github"
	"aireview/internal/llm"
	reportout "aireview/internal/report"
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
				if outputErr := writeReviewOutput(ref, report, err, opts); outputErr != nil {
					return outputErr
				}
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

func writeReviewOutput(ref github.PRRef, report review.ReviewReport, warning error, opts reviewOptions) error {
	format := strings.ToLower(strings.TrimSpace(opts.format))
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "markdown" {
		return fmt.Errorf("unsupported output format %q: use text or markdown", opts.format)
	}

	if opts.output != "" {
		var markdown bytes.Buffer
		if err := reportout.WriteMarkdown(&markdown, ref, report, warning); err != nil {
			return err
		}
		if err := os.WriteFile(opts.output, markdown.Bytes(), 0o644); err != nil {
			return fmt.Errorf("write Markdown report %s: %w", opts.output, err)
		}
	}

	if format == "markdown" {
		if opts.output != "" {
			fmt.Fprintf(os.Stdout, "Markdown report written to %s\n", opts.output)
			return nil
		}
		return reportout.WriteMarkdown(os.Stdout, ref, report, warning)
	}
	return reportout.WriteTerminal(os.Stdout, ref, report, warning)
}
