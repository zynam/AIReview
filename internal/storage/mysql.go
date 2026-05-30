package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"aireview/internal/review"
	"aireview/internal/session"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MySQLStore struct {
	db *gorm.DB
}

type reviewSessionRecord struct {
	ID               string         `gorm:"primaryKey;size:36"`
	ReviewerID       string         `gorm:"size:128;not null;index:idx_reviewer_created,priority:1"`
	Owner            string         `gorm:"size:128;not null;index:idx_repo_pr,priority:1"`
	Repo             string         `gorm:"size:128;not null;index:idx_repo_pr,priority:2"`
	PRNumber         int            `gorm:"not null;index:idx_repo_pr,priority:3"`
	HeadSHA          string         `gorm:"size:64;not null;default:''"`
	Status           session.Status `gorm:"size:32;not null;index:idx_status_updated,priority:1"`
	Summary          string         `gorm:"type:text"`
	ImpactJSON       string         `gorm:"column:impact_json;type:json"`
	TestAssessment   string         `gorm:"type:text"`
	SkippedFilesJSON string         `gorm:"column:skipped_files_json;type:json"`
	Error            string         `gorm:"type:text"`
	CreatedAt        time.Time      `gorm:"not null;index:idx_reviewer_created,priority:2"`
	UpdatedAt        time.Time      `gorm:"not null;index:idx_status_updated,priority:2"`
}

func (reviewSessionRecord) TableName() string {
	return "review_sessions"
}

type findingRecord struct {
	ID              string          `gorm:"primaryKey;size:36"`
	SessionID       string          `gorm:"size:36;not null;index:idx_session_severity,priority:1;index:idx_file_line,priority:1"`
	Severity        review.Severity `gorm:"size:16;not null;index:idx_session_severity,priority:2"`
	Confidence      float64         `gorm:"type:decimal(5,4);not null;default:0;index:idx_session_severity,priority:3"`
	Category        review.Category `gorm:"size:64;not null;default:''"`
	File            string          `gorm:"size:512;not null;default:'';index:idx_file_line,priority:2"`
	Line            int             `gorm:"not null;default:0;index:idx_file_line,priority:3"`
	Title           string          `gorm:"size:512;not null"`
	Evidence        string          `gorm:"type:text"`
	Suggestion      string          `gorm:"type:text"`
	NeedsHumanCheck bool            `gorm:"not null;default:0"`
	FeedbackStatus  string          `gorm:"size:32;not null;default:''"`
	CreatedAt       time.Time       `gorm:"not null"`
}

func (findingRecord) TableName() string {
	return "findings"
}

type contextChunkRecord struct {
	ID        string `gorm:"primaryKey;size:36"`
	SessionID string `gorm:"size:36;not null;index:idx_session_file_kind,priority:1"`
	File      string `gorm:"size:512;not null;default:'';index:idx_session_file_kind,priority:2"`
	Kind      string `gorm:"size:64;not null;index:idx_session_file_kind,priority:3"`
	Content   string `gorm:"type:mediumtext"`
	Tokens    int    `gorm:"not null;default:0"`
	Score     float64
	CreatedAt time.Time `gorm:"not null"`
}

func (contextChunkRecord) TableName() string {
	return "context_chunks"
}

type reviewEventRecord struct {
	ID        string    `gorm:"primaryKey;size:36"`
	SessionID string    `gorm:"size:36;not null;index:idx_session_created,priority:1"`
	Type      string    `gorm:"size:64;not null"`
	Message   string    `gorm:"type:text"`
	CreatedAt time.Time `gorm:"not null;index:idx_session_created,priority:2"`
}

func (reviewEventRecord) TableName() string {
	return "review_events"
}

func OpenMySQL(ctx context.Context, dsn string) (*gorm.DB, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("mysql dsn is required")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get mysql db handle: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return db, nil
}

func NewMySQLStore(db *gorm.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) CreateSession(ctx context.Context, item session.ReviewSession) error {
	record := sessionToRecord(item)
	if record.ID == "" {
		record.ID = session.NewID()
	}
	now := time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	if record.Status == "" {
		record.Status = session.StatusCreated
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return fmt.Errorf("create review session: %w", err)
	}
	return nil
}

func (s *MySQLStore) GetSession(ctx context.Context, id string) (session.ReviewSession, error) {
	var record reviewSessionRecord
	if err := s.db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
		return session.ReviewSession{}, mapNotFound(err)
	}
	item := recordToSession(record)
	if err := s.db.WithContext(ctx).Model(&findingRecord{}).Where("session_id = ?", id).Count(&item.FindingsCount).Error; err != nil {
		return session.ReviewSession{}, fmt.Errorf("count findings: %w", err)
	}
	return item, nil
}

func (s *MySQLStore) ListSessions(ctx context.Context, filter session.ListFilter) ([]session.ReviewSession, error) {
	query := s.db.WithContext(ctx).Model(&reviewSessionRecord{})
	if filter.ReviewerID != "" {
		query = query.Where("reviewer_id = ?", filter.ReviewerID)
	}
	if filter.Owner != "" {
		query = query.Where("owner = ?", filter.Owner)
	}
	if filter.Repo != "" {
		query = query.Where("repo = ?", filter.Repo)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	var records []reviewSessionRecord
	if err := query.Order("created_at desc").Limit(100).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list review sessions: %w", err)
	}

	items := make([]session.ReviewSession, 0, len(records))
	for _, record := range records {
		item := recordToSession(record)
		if err := s.db.WithContext(ctx).Model(&findingRecord{}).Where("session_id = ?", record.ID).Count(&item.FindingsCount).Error; err != nil {
			return nil, fmt.Errorf("count findings: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *MySQLStore) UpdateStatus(ctx context.Context, id string, status session.Status, message string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&reviewSessionRecord{}).Where("id = ?", id).Updates(map[string]any{
			"status":     status,
			"error":      message,
			"updated_at": time.Now(),
		})
		if result.Error != nil {
			return fmt.Errorf("update review status: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return session.ErrNotFound
		}

		event := reviewEventRecord{
			ID:        session.NewID(),
			SessionID: id,
			Type:      string(status),
			Message:   message,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&event).Error; err != nil {
			return fmt.Errorf("insert review event: %w", err)
		}
		return nil
	})
}

func (s *MySQLStore) SaveReport(ctx context.Context, id string, report review.ReviewReport) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&reviewSessionRecord{}).Where("id = ?", id).Updates(map[string]any{
			"summary":            report.Summary,
			"impact_json":        jsonText(report.Impact),
			"test_assessment":    report.TestAssessment,
			"skipped_files_json": jsonText(report.SkippedFiles),
			"error":              "",
			"updated_at":         time.Now(),
		})
		if result.Error != nil {
			return fmt.Errorf("update review report: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return session.ErrNotFound
		}
		if err := tx.Where("session_id = ?", id).Delete(&findingRecord{}).Error; err != nil {
			return fmt.Errorf("replace findings: %w", err)
		}

		records := make([]findingRecord, 0, len(report.Findings))
		now := time.Now()
		for _, finding := range report.Findings {
			if finding.ID == "" {
				finding.ID = session.NewID()
			}
			records = append(records, findingToRecord(id, finding, now))
		}
		if len(records) > 0 {
			if err := tx.Create(&records).Error; err != nil {
				return fmt.Errorf("insert findings: %w", err)
			}
		}
		return nil
	})
}

func (s *MySQLStore) ListFindings(ctx context.Context, sessionID string) ([]review.Finding, error) {
	var records []findingRecord
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("field(severity, 'high', 'medium', 'low') asc, confidence desc, file asc, line asc").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}

	findings := make([]review.Finding, 0, len(records))
	for _, record := range records {
		findings = append(findings, recordToFinding(record))
	}
	return findings, nil
}

func (s *MySQLStore) ListContexts(ctx context.Context, sessionID string) ([]session.ContextChunk, error) {
	var records []contextChunkRecord
	if err := s.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("score desc, created_at asc").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("list context chunks: %w", err)
	}

	chunks := make([]session.ContextChunk, 0, len(records))
	for _, record := range records {
		chunks = append(chunks, session.ContextChunk{
			ID:        record.ID,
			SessionID: record.SessionID,
			File:      record.File,
			Kind:      record.Kind,
			Content:   record.Content,
			Tokens:    record.Tokens,
			Score:     record.Score,
			CreatedAt: record.CreatedAt,
		})
	}
	return chunks, nil
}

func (s *MySQLStore) UpdateFindingFeedback(ctx context.Context, findingID string, status string) error {
	result := s.db.WithContext(ctx).Model(&findingRecord{}).Where("id = ?", findingID).Update("feedback_status", status)
	if result.Error != nil {
		return fmt.Errorf("update finding feedback: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return session.ErrNotFound
	}
	return nil
}

func sessionToRecord(item session.ReviewSession) reviewSessionRecord {
	return reviewSessionRecord{
		ID:               item.ID,
		ReviewerID:       item.ReviewerID,
		Owner:            item.Owner,
		Repo:             item.Repo,
		PRNumber:         item.PRNumber,
		HeadSHA:          item.HeadSHA,
		Status:           item.Status,
		Summary:          item.Summary,
		ImpactJSON:       jsonText(item.Impact),
		TestAssessment:   item.TestAssessment,
		SkippedFilesJSON: jsonText(item.SkippedFiles),
		Error:            item.Error,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func recordToSession(record reviewSessionRecord) session.ReviewSession {
	return session.ReviewSession{
		ID:             record.ID,
		ReviewerID:     record.ReviewerID,
		Owner:          record.Owner,
		Repo:           record.Repo,
		PRNumber:       record.PRNumber,
		HeadSHA:        record.HeadSHA,
		Status:         record.Status,
		Summary:        record.Summary,
		Impact:         stringSlice(record.ImpactJSON),
		TestAssessment: record.TestAssessment,
		SkippedFiles:   stringSlice(record.SkippedFilesJSON),
		Error:          record.Error,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func findingToRecord(sessionID string, finding review.Finding, createdAt time.Time) findingRecord {
	return findingRecord{
		ID:              finding.ID,
		SessionID:       sessionID,
		Severity:        finding.Severity,
		Confidence:      finding.Confidence,
		Category:        finding.Category,
		File:            finding.File,
		Line:            finding.Line,
		Title:           finding.Title,
		Evidence:        finding.Evidence,
		Suggestion:      finding.Suggestion,
		NeedsHumanCheck: finding.NeedsHumanCheck,
		FeedbackStatus:  finding.FeedbackStatus,
		CreatedAt:       createdAt,
	}
}

func recordToFinding(record findingRecord) review.Finding {
	return review.Finding{
		ID:              record.ID,
		Severity:        record.Severity,
		Confidence:      record.Confidence,
		Category:        record.Category,
		File:            record.File,
		Line:            record.Line,
		Title:           record.Title,
		Evidence:        record.Evidence,
		Suggestion:      record.Suggestion,
		NeedsHumanCheck: record.NeedsHumanCheck,
		FeedbackStatus:  record.FeedbackStatus,
	}
}

func jsonText(values []string) string {
	if values == nil {
		values = []string{}
	}
	body, _ := json.Marshal(values)
	return string(body)
}

func stringSlice(body string) []string {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(body), &values); err != nil {
		return nil
	}
	return values
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return session.ErrNotFound
	}
	return fmt.Errorf("get review session: %w", err)
}
