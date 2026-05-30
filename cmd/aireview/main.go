package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"

	reviewagent "aireview/internal/agent"
	"aireview/internal/app"
	"aireview/internal/config"
	"aireview/internal/github"
	"aireview/internal/jobs"
	reportout "aireview/internal/report"
	"aireview/internal/review"
	"aireview/internal/server"
	"aireview/internal/session"
	"aireview/internal/storage"

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

type serverOptions struct {
	port       int
	mysqlDSN   string
	configPath string
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
	cmd.AddCommand(newServerCommand())
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
				Agent:  reviewagent.NewEinoReviewAgent(),
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

func newServerCommand() *cobra.Command {
	var opts serverOptions

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the AIReview Web API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(opts.configPath)
			if err != nil {
				return err
			}
			dsn := strings.TrimSpace(opts.mysqlDSN)
			if dsn == "" {
				dsn = strings.TrimSpace(os.Getenv("MYSQL_DSN"))
			}
			if dsn == "" {
				dsn = strings.TrimSpace(cfg.MySQL.DSN)
			}
			if dsn == "" {
				router := server.NewRouter(server.Dependencies{})
				addr := fmt.Sprintf(":%d", opts.port)
				httpServer := &http.Server{
					Addr:    addr,
					Handler: router,
				}
				return httpServer.ListenAndServe()
			}
			db, err := storage.OpenMySQL(cmd.Context(), dsn)
			if err != nil {
				return err
			}
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			defer sqlDB.Close()
			if err := storage.Migrate(cmd.Context(), db); err != nil {
				return err
			}

			store := storage.NewMySQLStore(db)
			queue := jobs.NewQueue(100)
			reviewService := &app.ReviewService{
				GitHub: github.NewClient(),
				Agent:  reviewagent.NewEinoReviewAgent(),
			}
			worker := jobs.Worker{
				Queue:         queue,
				Store:         store,
				ReviewService: reviewService,
			}
			go worker.Run(cmd.Context())

			sessionService := session.Service{Store: store}
			router := server.NewRouter(server.Dependencies{
				Store:   store,
				Queue:   queue,
				Config:  cfg,
				Service: sessionService,
			})

			addr := fmt.Sprintf(":%d", opts.port)
			httpServer := &http.Server{
				Addr:    addr,
				Handler: router,
			}
			return httpServer.ListenAndServe()
		},
	}

	cmd.Flags().IntVar(&opts.port, "port", 8080, "HTTP server port")
	cmd.Flags().StringVar(&opts.mysqlDSN, "mysql-dsn", "", "MySQL DSN, defaults to MYSQL_DSN")
	cmd.Flags().StringVar(&opts.configPath, "config", "", "path to .aireview.toml")

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
