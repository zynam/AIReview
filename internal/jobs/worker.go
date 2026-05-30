package jobs

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"aireview/internal/app"
	"aireview/internal/session"
)

type Worker struct {
	Queue         *Queue
	Store         session.Store
	ReviewService *app.ReviewService
	Timeout       time.Duration
}

func (w *Worker) Run(ctx context.Context) {
	if w.Timeout <= 0 {
		w.Timeout = 10 * time.Minute
	}

	for {
		select {
		case <-ctx.Done():
			return
		case job := <-w.Queue.Jobs():
			w.process(ctx, job)
		}
	}
}

func (w *Worker) process(parent context.Context, job ReviewJob) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message := fmt.Sprintf("review worker panic: %v", recovered)
			log.Printf("%s\n%s", message, debug.Stack())
			_ = w.Store.UpdateStatus(context.Background(), job.SessionID, session.StatusFailed, message)
		}
	}()

	ctx, cancel := context.WithTimeout(parent, w.Timeout)
	defer cancel()

	if err := w.Store.UpdateStatus(ctx, job.SessionID, session.StatusFetchingPR, ""); err != nil {
		log.Printf("update review status to fetching_pr: %v", err)
		return
	}
	if err := w.Store.UpdateStatus(ctx, job.SessionID, session.StatusAnalyzing, ""); err != nil {
		log.Printf("update review status to analyzing: %v", err)
		return
	}

	result, err := w.ReviewService.ReviewPRWithDetails(ctx, app.ReviewPRRequest{
		Ref:      job.Ref,
		Config:   job.Config,
		MaxFiles: job.MaxFiles,
	})
	if err != nil {
		_ = w.Store.UpdateStatus(context.Background(), job.SessionID, session.StatusFailed, err.Error())
		return
	}
	if result.PullRequest.HeadSHA != "" {
		if err := w.Store.UpdateHeadSHA(ctx, job.SessionID, result.PullRequest.HeadSHA); err != nil {
			_ = w.Store.UpdateStatus(context.Background(), job.SessionID, session.StatusFailed, err.Error())
			return
		}
	}
	if err := w.Store.SaveReport(ctx, job.SessionID, result.Report); err != nil {
		_ = w.Store.UpdateStatus(context.Background(), job.SessionID, session.StatusFailed, err.Error())
		return
	}
	if err := w.Store.UpdateStatus(ctx, job.SessionID, session.StatusCompleted, ""); err != nil {
		log.Printf("update review status to completed: %v", err)
	}
}
