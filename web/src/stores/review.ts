import { defineStore } from "pinia";

import { getReview, updateFindingFeedback } from "@/api/reviews";
import type {
  Finding,
  FindingFeedbackStatus,
  ReviewDetail,
  ReviewStatus,
} from "@/api/types";

const terminalStatuses: ReviewStatus[] = ["completed", "failed", "cancelled"];

export const useReviewStore = defineStore("review", {
  state: () => ({
    current: null as ReviewDetail | null,
    findings: [] as Finding[],
    loading: false,
    error: "",
    pollingTimer: null as ReturnType<typeof setInterval> | null,
  }),
  getters: {
    isTerminal: (state) =>
      state.current ? terminalStatuses.includes(state.current.status) : false,
  },
  actions: {
    async fetchReview(id: string) {
      this.loading = true;
      this.error = "";
      try {
        const detail = await getReview(id);
        this.current = detail;
        this.findings = detail.findings ?? [];
        if (terminalStatuses.includes(detail.status)) {
          this.stopPolling();
        }
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Failed to load review";
      } finally {
        this.loading = false;
      }
    },
    startPolling(id: string) {
      this.stopPolling();
      this.pollingTimer = setInterval(() => {
        if (!this.isTerminal) {
          void this.fetchReview(id);
        }
      }, 2000);
    },
    stopPolling() {
      if (this.pollingTimer) {
        clearInterval(this.pollingTimer);
        this.pollingTimer = null;
      }
    },
    async setFindingFeedback(id: string, status: FindingFeedbackStatus) {
      await updateFindingFeedback(id, status);
      this.findings = this.findings.map((finding) =>
        finding.id === id ? { ...finding, feedback_status: status } : finding,
      );
      if (this.current) {
        this.current.findings = this.findings;
      }
    },
    reset() {
      this.stopPolling();
      this.current = null;
      this.findings = [];
      this.loading = false;
      this.error = "";
    },
  },
});
