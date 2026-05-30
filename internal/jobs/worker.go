package jobs

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"aireview/internal/agent"
	"aireview/internal/app"
	"aireview/internal/reviewcontext"
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

	result, err := w.ReviewService.ReviewPRWithDetails(ctx, app.ReviewPRRequest{
		Ref:       job.Ref,
		Config:    job.Config,
		MaxFiles:  job.MaxFiles,
		EventSink: agent.SessionEventSink{SessionID: job.SessionID, Sink: storeEventSink{Store: w.Store}},
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
	if err := w.Store.SaveAgentArtifacts(ctx, job.SessionID, session.AgentArtifacts{
		ContextChunks: contextChunks(result.Agent.Context.CompressedContext),
		Metrics:       result.Agent.Metrics,
	}); err != nil {
		_ = w.Store.UpdateStatus(context.Background(), job.SessionID, session.StatusFailed, err.Error())
		return
	}
	if err := w.Store.UpdateStatus(ctx, job.SessionID, session.StatusCompleted, ""); err != nil {
		log.Printf("update review status to completed: %v", err)
	}
}

type storeEventSink struct {
	Store session.Store
}

func (s storeEventSink) Emit(ctx context.Context, event agent.Event) error {
	if event.SessionID == "" || s.Store == nil {
		return nil
	}
	if err := s.Store.SaveReviewEvent(ctx, session.ReviewEvent{
		SessionID: event.SessionID,
		Type:      event.Type,
		Message:   event.Message,
		CreatedAt: event.Time,
	}); err != nil {
		return err
	}
	switch event.Type {
	case agent.EventContextBuildStarted:
		return s.Store.UpdateStatus(ctx, event.SessionID, session.StatusBuildingContext, "")
	case agent.EventLLMCallStarted:
		return s.Store.UpdateStatus(ctx, event.SessionID, session.StatusAnalyzing, "")
	default:
		return nil
	}
}

func contextChunks(ctx reviewcontext.ReviewContext) []session.ContextChunk {
	chunks := make([]session.ContextChunk, 0, len(ctx.Chunks))
	now := time.Now()
	for _, chunk := range ctx.Chunks {
		chunks = append(chunks, session.ContextChunk{
			ID:        session.NewID(),
			File:      chunk.File,
			Kind:      string(chunk.Kind),
			Content:   chunk.Content,
			Tokens:    chunk.Tokens,
			Score:     chunk.Score,
			CreatedAt: now,
		})
	}
	return chunks
}
