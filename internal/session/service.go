package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"aireview/internal/github"
)

type Service struct {
	Store Store
}

type CreateSessionRequest struct {
	PRURL      string
	ReviewerID string
}

type CreateSessionResult struct {
	Session ReviewSession
	Ref     github.PRRef
}

func (s Service) Create(ctx context.Context, req CreateSessionRequest) (CreateSessionResult, error) {
	ref, err := github.ParsePRURL(req.PRURL)
	if err != nil {
		return CreateSessionResult{}, err
	}
	reviewerID := strings.TrimSpace(req.ReviewerID)
	if reviewerID == "" {
		return CreateSessionResult{}, fmt.Errorf("reviewer_id is required")
	}

	now := time.Now()
	item := ReviewSession{
		ID:         NewID(),
		ReviewerID: reviewerID,
		Owner:      ref.Owner,
		Repo:       ref.Repo,
		PRNumber:   ref.Number,
		Status:     StatusCreated,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := s.Store.CreateSession(ctx, item); err != nil {
		return CreateSessionResult{}, err
	}
	return CreateSessionResult{Session: item, Ref: ref}, nil
}

func (s Service) Detail(ctx context.Context, id string) (ReviewDetail, error) {
	item, err := s.Store.GetSession(ctx, id)
	if err != nil {
		return ReviewDetail{}, err
	}
	findings, err := s.Store.ListFindings(ctx, id)
	if err != nil {
		return ReviewDetail{}, err
	}
	return ReviewDetail{ReviewSession: item, Findings: findings}, nil
}

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(b[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
