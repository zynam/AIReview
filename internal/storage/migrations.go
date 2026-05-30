package storage

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

var migrations = []string{
	`create table if not exists review_sessions (
  id varchar(36) primary key,
  reviewer_id varchar(128) not null,
  owner varchar(128) not null,
  repo varchar(128) not null,
  pr_number int not null,
  head_sha varchar(64) not null default '',
  status varchar(32) not null,
  summary text,
  impact_json json,
  test_assessment text,
  skipped_files_json json,
  error text,
  created_at datetime(3) not null,
  updated_at datetime(3) not null,
  index idx_reviewer_created (reviewer_id, created_at),
  index idx_repo_pr (owner, repo, pr_number),
  index idx_status_updated (status, updated_at)
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci`,
	`create table if not exists findings (
  id varchar(36) primary key,
  session_id varchar(36) not null,
  severity varchar(16) not null,
  confidence decimal(5,4) not null default 0,
  category varchar(64) not null default '',
  file varchar(512) not null default '',
  line int not null default 0,
  title varchar(512) not null,
  evidence text,
  suggestion text,
  needs_human_check tinyint(1) not null default 0,
  feedback_status varchar(32) not null default '',
  created_at datetime(3) not null,
  index idx_session_severity (session_id, severity, confidence),
  index idx_file_line (session_id, file, line),
  constraint fk_findings_session foreign key (session_id) references review_sessions(id) on delete cascade
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci`,
	`create table if not exists context_chunks (
  id varchar(36) primary key,
  session_id varchar(36) not null,
  file varchar(512) not null default '',
  kind varchar(64) not null,
  content mediumtext,
  tokens int not null default 0,
  score double not null default 0,
  created_at datetime(3) not null,
  index idx_session_file_kind (session_id, file, kind),
  constraint fk_context_chunks_session foreign key (session_id) references review_sessions(id) on delete cascade
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci`,
	`create table if not exists review_events (
  id varchar(36) primary key,
  session_id varchar(36) not null,
  type varchar(64) not null,
  message text,
  created_at datetime(3) not null,
  index idx_session_created (session_id, created_at),
  constraint fk_review_events_session foreign key (session_id) references review_sessions(id) on delete cascade
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci`,
	`create table if not exists llm_calls (
  id varchar(36) primary key,
  session_id varchar(36) not null,
  model varchar(128) not null default '',
  file_count int not null default 0,
  rule_finding_count int not null default 0,
  context_chunks int not null default 0,
  kept_chunks int not null default 0,
  skipped_files int not null default 0,
  prompt_tokens_approx int not null default 0,
  request_bytes int not null default 0,
  duration_millis bigint not null default 0,
  created_at datetime(3) not null,
  index idx_session_created (session_id, created_at),
  constraint fk_llm_calls_session foreign key (session_id) references review_sessions(id) on delete cascade
) engine=InnoDB default charset=utf8mb4 collate=utf8mb4_unicode_ci`,
}

func Migrate(ctx context.Context, db *gorm.DB) error {
	for i, statement := range migrations {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			return fmt.Errorf("run migration %d: %w", i+1, err)
		}
	}
	return nil
}
