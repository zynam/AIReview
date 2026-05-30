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
  llm_calls?: LLMCall[];
}

export interface ContextChunk {
  id: string;
  file: string;
  kind: string;
  content: string;
  tokens: number;
  score: number;
}

export interface ReviewEvent {
  id: string;
  session_id: string;
  type: string;
  message: string;
  created_at: string;
}

export interface LLMCall {
  id: string;
  session_id: string;
  model: string;
  file_count: number;
  rule_finding_count: number;
  context_chunks: number;
  kept_chunks: number;
  skipped_files: number;
  prompt_tokens_approx: number;
  request_bytes: number;
  duration_millis: number;
  created_at: string;
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
