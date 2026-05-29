package main

import (
	"fmt"
	"os"

	"aireview/internal/github"
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
			printPRSummary(pr)
			return nil
		},
	}

	cmd.Flags().StringVar(&opts.owner, "owner", "", "GitHub repository owner")
	cmd.Flags().StringVar(&opts.repo, "repo", "", "GitHub repository name")
	cmd.Flags().IntVar(&opts.prNumber, "pr", 0, "GitHub pull request number")
	cmd.Flags().StringVar(&opts.output, "output", "", "write Markdown report to this file")
	cmd.Flags().StringVar(&opts.format, "format", "text", "output format: text or markdown")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .aireview.yml")
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

func printPRSummary(pr review.PullRequest) {
	fmt.Printf("PR #%d %s/%s\n", pr.Number, pr.Owner, pr.Repo)
	fmt.Printf("Title: %s\n", pr.Title)
	fmt.Printf("Author: %s\n", pr.Author)
	fmt.Printf("Base: %s\n", pr.BaseSHA)
	fmt.Printf("Head: %s\n", pr.HeadSHA)
	fmt.Printf("Changed files: %d\n", len(pr.Files))
	for _, file := range pr.Files {
		fmt.Printf("- %s (%s, +%d -%d)\n", file.Path, file.Status, file.Additions, file.Deletions)
	}
	fmt.Printf("Commits: %d\n", len(pr.Commits))
	for _, commit := range pr.Commits {
		fmt.Printf("- %s %s (%s)\n", shortSHA(commit.SHA), firstLine(commit.Message), commit.Author)
	}
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
