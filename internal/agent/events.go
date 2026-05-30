package agent

import (
	"context"
	"time"
)

const (
	EventContextBuildStarted     = "context_build_started"
	EventContextBuildCompleted   = "context_build_completed"
	EventContextRetrieveComplete = "context_retrieve_completed"
	EventContextCompressComplete = "context_compress_completed"
	EventLLMCallStarted          = "llm_call_started"
	EventLLMCallCompleted        = "llm_call_completed"
	EventParseCompleted          = "parse_completed"
	EventValidateCompleted       = "validate_completed"
	EventMergeCompleted          = "merge_completed"
	EventReviewCompleted         = "review_completed"
)

type Event struct {
	SessionID string
	Type      string
	Message   string
	Time      time.Time
}

type EventSink interface {
	Emit(ctx context.Context, event Event) error
}

type SessionEventSink struct {
	SessionID string
	Sink      EventSink
}

func (s SessionEventSink) Emit(ctx context.Context, event Event) error {
	if s.Sink == nil {
		return nil
	}
	if event.SessionID == "" {
		event.SessionID = s.SessionID
	}
	return s.Sink.Emit(ctx, event)
}

func emit(ctx context.Context, sink EventSink, eventType string, message string) error {
	if sink == nil {
		return nil
	}
	return sink.Emit(ctx, Event{
		Type:    eventType,
		Message: message,
		Time:    time.Now(),
	})
}
