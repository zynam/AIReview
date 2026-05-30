package jobs

import (
	"context"
	"errors"

	"aireview/internal/config"
	"aireview/internal/github"
)

var ErrQueueFull = errors.New("review job queue is full")

type ReviewJob struct {
	SessionID string
	Ref       github.PRRef
	Config    config.Config
	MaxFiles  int
}

type Queue struct {
	jobs chan ReviewJob
}

func NewQueue(size int) *Queue {
	if size <= 0 {
		size = 100
	}
	return &Queue{jobs: make(chan ReviewJob, size)}
}

func (q *Queue) Enqueue(ctx context.Context, job ReviewJob) error {
	select {
	case q.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

func (q *Queue) Jobs() <-chan ReviewJob {
	return q.jobs
}
