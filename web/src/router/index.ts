import { createRouter, createWebHistory } from "vue-router";

const ReviewListPage = () => import("@/pages/ReviewListPage.vue");
const NewReviewPage = () => import("@/pages/NewReviewPage.vue");
const ReviewDetailPage = () => import("@/pages/ReviewDetailPage.vue");

export default createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/", redirect: "/reviews" },
    { path: "/reviews", component: ReviewListPage },
    { path: "/reviews/new", component: NewReviewPage },
    { path: "/reviews/:id", component: ReviewDetailPage },
  ],
});
