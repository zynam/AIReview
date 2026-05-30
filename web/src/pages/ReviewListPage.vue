<template>
  <section class="page-stack">
    <div class="panel">
      <div class="panel-header">
        <div>
          <h2>Review Sessions</h2>
          <p>Filter by reviewer, repository, or current status.</p>
        </div>
        <el-button type="primary" :icon="Plus" @click="router.push('/reviews/new')">
          New Review
        </el-button>
      </div>

      <el-form :model="filters" class="filter-bar" inline @submit.prevent>
        <el-form-item label="Reviewer">
          <el-input
            v-model="filters.reviewer_id"
            clearable
            placeholder="alice"
            @keyup.enter="loadReviews"
          />
        </el-form-item>
        <el-form-item label="Repo">
          <el-input
            v-model="filters.repo"
            clearable
            placeholder="org/repo"
            @keyup.enter="loadReviews"
          />
        </el-form-item>
        <el-form-item label="Status">
          <el-select v-model="filters.status" clearable placeholder="Any status">
            <el-option label="Created" value="created" />
            <el-option label="Fetching PR" value="fetching_pr" />
            <el-option label="Building Context" value="building_context" />
            <el-option label="Analyzing" value="analyzing" />
            <el-option label="Completed" value="completed" />
            <el-option label="Failed" value="failed" />
            <el-option label="Cancelled" value="cancelled" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button :icon="Search" :loading="loading" type="primary" @click="loadReviews">
            Search
          </el-button>
          <el-button :icon="Refresh" @click="resetFilters">Reset</el-button>
        </el-form-item>
      </el-form>

      <el-alert
        v-if="error"
        class="section-alert"
        :title="error"
        type="error"
        show-icon
        :closable="false"
      />

      <el-table
        v-loading="loading"
        :data="reviews"
        row-key="id"
        border
        empty-text="No reviews found"
        @row-click="openReview"
      >
        <el-table-column label="PR" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ formatPR(row) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="reviewer_id" label="Reviewer" width="150" />
        <el-table-column label="Status" width="150">
          <template #default="{ row }">
            <ReviewStatusTag :status="row.status" />
          </template>
        </el-table-column>
        <el-table-column prop="findings_count" label="Findings" width="110" />
        <el-table-column label="Created" width="190">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="Updated" width="190">
          <template #default="{ row }">{{ formatDate(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-button :icon="View" link type="primary" @click.stop="openReview(row)">
              Open
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Plus, Refresh, Search, View } from "@element-plus/icons-vue";
import { onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";

import { listReviews } from "@/api/reviews";
import type { ListReviewsParams, ReviewSession } from "@/api/types";
import ReviewStatusTag from "@/components/ReviewStatusTag.vue";

const router = useRouter();
const reviews = ref<ReviewSession[]>([]);
const loading = ref(false);
const error = ref("");
const filters = reactive<ListReviewsParams>({
  reviewer_id: "",
  repo: "",
  status: "",
});

async function loadReviews() {
  loading.value = true;
  error.value = "";
  try {
    reviews.value = await listReviews({
      reviewer_id: filters.reviewer_id?.trim() || undefined,
      repo: filters.repo?.trim() || undefined,
      status: filters.status || undefined,
    });
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load reviews";
    reviews.value = [];
  } finally {
    loading.value = false;
  }
}

function resetFilters() {
  filters.reviewer_id = "";
  filters.repo = "";
  filters.status = "";
  void loadReviews();
}

function openReview(row: ReviewSession) {
  void router.push(`/reviews/${row.id}`);
}

function formatPR(row: ReviewSession) {
  const repo = row.owner && row.repo ? `${row.owner}/${row.repo}` : "unknown/repo";
  return `${repo}#${row.pr_number || "-"}`;
}

function formatDate(value: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

onMounted(loadReviews);
</script>
