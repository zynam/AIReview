import { defineStore } from "pinia";

const reviewerKey = "aireview.reviewer_id";

export const useUserStore = defineStore("user", {
  state: () => ({
    reviewerId: localStorage.getItem(reviewerKey) ?? "",
  }),
  actions: {
    setReviewerId(value: string) {
      this.reviewerId = value.trim();
      localStorage.setItem(reviewerKey, this.reviewerId);
    },
  },
});
