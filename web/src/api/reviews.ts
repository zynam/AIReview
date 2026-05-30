import { http } from "./client";
import type {
  ContextChunk,
  CreateReviewPayload,
  CreateReviewResponse,
  FindingFeedbackStatus,
  ListReviewsParams,
  ReviewDetail,
  ReviewEvent,
  ReviewSession,
} from "./types";

interface ListResponse<T> {
  items: T[];
}

export async function listReviews(
  params: ListReviewsParams = {},
): Promise<ReviewSession[]> {
  const { data } = await http.get<ListResponse<ReviewSession> | ReviewSession[]>(
    "/api/v1/reviews",
    { params },
  );
  return Array.isArray(data) ? data : data.items ?? [];
}

export async function createReview(
  payload: CreateReviewPayload,
): Promise<CreateReviewResponse> {
  const { data } = await http.post<CreateReviewResponse>("/api/v1/reviews", payload);
  return data;
}

export async function getReview(id: string): Promise<ReviewDetail> {
  const { data } = await http.get<ReviewDetail>(`/api/v1/reviews/${id}`);
  return {
    ...data,
    findings: data.findings ?? [],
    skipped_files: data.skipped_files ?? [],
    llm_calls: data.llm_calls ?? [],
    impact: data.impact ?? [],
    findings_count: data.findings_count ?? data.findings?.length ?? 0,
    test_assessment: data.test_assessment ?? "",
    summary: data.summary ?? "",
  };
}

export async function getContexts(id: string): Promise<ContextChunk[]> {
  const { data } = await http.get<ListResponse<ContextChunk> | ContextChunk[]>(
    `/api/v1/reviews/${id}/contexts`,
  );
  return Array.isArray(data) ? data : data.items ?? [];
}

export async function getEvents(id: string): Promise<ReviewEvent[]> {
  const { data } = await http.get<ListResponse<ReviewEvent> | ReviewEvent[]>(
    `/api/v1/reviews/${id}/events`,
  );
  return Array.isArray(data) ? data : data.items ?? [];
}

export async function updateFindingFeedback(
  id: string,
  feedbackStatus: FindingFeedbackStatus,
): Promise<void> {
  await http.patch(`/api/v1/findings/${id}`, {
    feedback_status: feedbackStatus,
  });
}

export function getReportURL(id: string): string {
  const baseURL = http.defaults.baseURL ?? window.location.origin;
  return new URL(`/api/v1/reviews/${id}/report.md`, baseURL).toString();
}
