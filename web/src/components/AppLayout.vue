<template>
  <el-container class="app-shell">
    <el-aside width="236px" class="side-nav">
      <div class="brand">
        <span class="brand-mark">AI</span>
        <span>PR Review</span>
      </div>
      <el-menu :default-active="activeMenu" router class="nav-menu">
        <el-menu-item index="/reviews">
          <el-icon><Tickets /></el-icon>
          <span>Reviews</span>
        </el-menu-item>
        <el-menu-item index="/reviews/new">
          <el-icon><Plus /></el-icon>
          <span>New Review</span>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container>
      <el-header class="top-bar">
        <div>
          <h1>{{ pageTitle }}</h1>
          <p>{{ pageSubtitle }}</p>
        </div>
        <ReviewStatusTag v-if="review.current" :status="review.current.status" />
      </el-header>
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { Plus, Tickets } from "@element-plus/icons-vue";
import { computed } from "vue";
import { useRoute } from "vue-router";

import ReviewStatusTag from "@/components/ReviewStatusTag.vue";
import { useReviewStore } from "@/stores/review";

const route = useRoute();
const review = useReviewStore();

const activeMenu = computed(() =>
  route.path.startsWith("/reviews/new") ? "/reviews/new" : "/reviews",
);

const pageTitle = computed(() => {
  if (route.path === "/reviews/new") return "New Review";
  if (route.params.id) return "Review Detail";
  return "Review Sessions";
});

const pageSubtitle = computed(() => {
  if (route.path === "/reviews/new") return "Create an AI-assisted PR analysis.";
  if (route.params.id) return "Inspect findings, context, and report output.";
  return "Search and open review sessions.";
});
</script>
