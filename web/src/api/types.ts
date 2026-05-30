export type ReviewStatus =
  | "created"
  | "fetching_pr"
  | "building_context"
  | "analyzing"
  | "completed"
  | "failed"
  | "cancelled";

export type FindingSeverity = "low" | "medium" | "high";

export type FindingFeedbackStatus =
  | "useful"
  | "false_positive"
  | "fixed"
  | "ignored";

export interface ReviewSession {
  id: string;
  reviewer_id: string;
  owner: string;
  repo: string;
  pr_number: number;
  head_sha: string;
  status: ReviewStatus;
  summary: string;
  impact: string[];
  test_assessment: string;
  findings_count: number;
  error?: string;
  created_at: string;
  updated_at: string;
}

export interface Finding {
  id: string;
  severity: FindingSeverity;
  confidence: number;
  category: string;
  file: string;
  line: number;
  title: string;
  evidence: string;
  suggestion: string;
  needs_human_check: boolean;
  feedback_status?: FindingFeedbackStatus;
}

export interface ReviewDetail extends ReviewSession {
  findings: Finding[];
  skipped_files: string[];
}

export interface ContextChunk {
  id: string;
  file: string;
  kind: string;
  content: string;
  tokens: number;
  score: number;
}

export interface ListReviewsParams {
  reviewer_id?: string;
  repo?: string;
  status?: ReviewStatus | "";
}

export interface CreateReviewPayload {
  pr_url: string;
  reviewer_id: string;
  config: {
    model: string;
    review_focus: string[];
    max_files: number;
  };
}

export interface CreateReviewResponse {
  id: string;
  status: ReviewStatus;
}
