<template>
  <section class="page-stack">
    <el-skeleton v-if="review.loading && !review.current" :rows="8" animated />

    <template v-else-if="review.current">
      <div class="panel">
        <div class="panel-header">
          <div>
            <h2>{{ review.current.owner }}/{{ review.current.repo }}#{{ review.current.pr_number }}</h2>
            <p class="mono">{{ review.current.head_sha || "No head SHA" }}</p>
          </div>
          <div class="toolbar">
            <ReviewStatusTag :status="review.current.status" />
            <el-button :icon="Refresh" :loading="review.loading" @click="reload">Refresh</el-button>
          </div>
        </div>
        <ReviewProgress :status="review.current.status" :error="review.current.error" />
      </div>

      <el-tabs v-model="activeTab" class="detail-tabs">
        <el-tab-pane label="Overview" name="overview">
          <div class="panel">
            <el-descriptions :column="3" border>
              <el-descriptions-item label="Reviewer">
                {{ review.current.reviewer_id || "-" }}
              </el-descriptions-item>
              <el-descriptions-item label="Findings">
                {{ review.findings.length }}
              </el-descriptions-item>
              <el-descriptions-item label="Updated">
                {{ formatDate(review.current.updated_at) }}
              </el-descriptions-item>
            </el-descriptions>

            <div class="overview-grid">
              <section>
                <h3>Summary</h3>
                <p>{{ review.current.summary || "Summary is not available yet." }}</p>
              </section>
              <section>
                <h3>Impact</h3>
                <el-empty v-if="review.current.impact.length === 0" description="No impact items" />
                <ul v-else class="plain-list">
                  <li v-for="item in review.current.impact" :key="item">{{ item }}</li>
                </ul>
              </section>
              <section>
                <h3>Test Assessment</h3>
                <p>
                  {{ review.current.test_assessment || "Test assessment is not available yet." }}
                </p>
              </section>
              <section>
                <h3>Skipped Files</h3>
                <el-empty
                  v-if="review.current.skipped_files.length === 0"
                  description="No skipped files"
                />
                <ul v-else class="plain-list mono">
                  <li v-for="file in review.current.skipped_files" :key="file">{{ file }}</li>
                </ul>
              </section>
            </div>
          </div>
        </el-tab-pane>

        <el-tab-pane label="Findings" name="findings">
          <div class="panel">
            <FindingsTable :findings="review.findings" @feedback="setFeedback" />
          </div>
        </el-tab-pane>

        <el-tab-pane label="Context" name="context" lazy>
          <ContextPanel :review-id="reviewId" />
        </el-tab-pane>

        <el-tab-pane label="Report" name="report" lazy>
          <ReportPreview :review-id="reviewId" />
        </el-tab-pane>
      </el-tabs>
    </template>

    <el-empty v-else description="Review not found">
      <el-button type="primary" @click="router.push('/reviews')">Back to Reviews</el-button>
    </el-empty>

    <el-alert
      v-if="review.error"
      class="section-alert"
      :title="review.error"
      type="error"
      show-icon
      :closable="false"
    />
  </section>
</template>

<script setup lang="ts">
import { Refresh } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

import type { FindingFeedbackStatus } from "@/api/types";
import ContextPanel from "@/components/ContextPanel.vue";
import FindingsTable from "@/components/FindingsTable.vue";
import ReportPreview from "@/components/ReportPreview.vue";
import ReviewProgress from "@/components/ReviewProgress.vue";
import ReviewStatusTag from "@/components/ReviewStatusTag.vue";
import { useReviewStore } from "@/stores/review";

const route = useRoute();
const router = useRouter();
const review = useReviewStore();
const activeTab = ref("overview");
const reviewId = computed(() => String(route.params.id));

async function reload() {
  await review.fetchReview(reviewId.value);
  if (!review.isTerminal) {
    review.startPolling(reviewId.value);
  }
}

async function setFeedback(id: string, status: FindingFeedbackStatus) {
  try {
    await review.setFindingFeedback(id, status);
    ElMessage.success("Feedback updated");
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : "Failed to update feedback");
  }
}

function formatDate(value: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

onMounted(reload);
watch(reviewId, reload);
onBeforeUnmount(() => review.reset());
</script>
