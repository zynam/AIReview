package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	reviewagent "aireview/internal/agent"
	"aireview/internal/app"
	"aireview/internal/config"
	"aireview/internal/github"
	"aireview/internal/jobs"
	"aireview/internal/server"
	"aireview/internal/session"
	"aireview/internal/storage"
)

type serverOptions struct {
	port       int
	mysqlDSN   string
	configPath string
}

func main() {
	opts := parseOptions()
	if err := run(context.Background(), opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseOptions() serverOptions {
	var opts serverOptions
	flag.IntVar(&opts.port, "port", 8080, "HTTP server port")
	flag.StringVar(&opts.mysqlDSN, "mysql-dsn", "", "MySQL DSN, defaults to MYSQL_DSN")
	flag.StringVar(&opts.configPath, "config", "configs/aireview.toml", "path to aireview TOML config")
	flag.Parse()
	return opts
}

func run(ctx context.Context, opts serverOptions) error {
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
		return fmt.Errorf("mysql dsn is required: set MYSQL_DSN, --mysql-dsn, or [mysql].dsn")
	}

	db, err := storage.OpenMySQL(ctx, dsn)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	if err := storage.Migrate(ctx, db); err != nil {
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
	go worker.Run(ctx)

	router := server.NewRouter(server.Dependencies{
		Store:   store,
		Queue:   queue,
		Config:  cfg,
		Service: session.Service{Store: store},
	})
	addr := fmt.Sprintf(":%d", opts.port)
	return (&http.Server{
		Addr:    addr,
		Handler: router,
	}).ListenAndServe()
}
