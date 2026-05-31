<template>
  <section class="page-stack">
    <el-skeleton v-if="review.loading && !review.current" :rows="8" animated />

    <template v-else-if="review.current">
      <div class="panel">
        <div class="panel-header">
          <div>
            <h2>{{ review.current.owner }}/{{ review.current.repo }}#{{ review.current.pr_number }}</h2>
            <p class="mono">{{ review.current.head_sha || "暂无 head SHA" }}</p>
          </div>
          <div class="toolbar">
            <ReviewStatusTag :status="review.current.status" />
            <el-button :icon="Refresh" :loading="review.loading" @click="reload">刷新</el-button>
          </div>
        </div>
        <ReviewProgress :status="review.current.status" :error="review.current.error" />
      </div>

      <el-tabs v-model="activeTab" class="detail-tabs">
        <el-tab-pane label="概览" name="overview">
          <div class="panel">
            <el-descriptions :column="3" border>
              <el-descriptions-item label="Reviewer">
                {{ review.current.reviewer_id || "-" }}
              </el-descriptions-item>
              <el-descriptions-item label="Findings">
                {{ review.findings.length }}
              </el-descriptions-item>
              <el-descriptions-item label="更新时间">
                {{ formatDate(review.current.updated_at) }}
              </el-descriptions-item>
              <el-descriptions-item label="模型">
                {{ latestLLMCall?.model || "-" }}
              </el-descriptions-item>
              <el-descriptions-item label="Prompt Tokens">
                {{ latestLLMCall?.prompt_tokens_approx ?? "-" }}
              </el-descriptions-item>
              <el-descriptions-item label="耗时">
                {{ formatDuration(latestLLMCall?.duration_millis) }}
              </el-descriptions-item>
            </el-descriptions>

            <div class="overview-grid">
              <section>
                <h3>Summary 摘要</h3>
                <p>{{ review.current.summary || "Summary 暂不可用。" }}</p>
              </section>
              <section>
                <h3>Impact 影响范围</h3>
                <el-empty v-if="review.current.impact.length === 0" description="暂无影响项" />
                <ul v-else class="plain-list">
                  <li v-for="item in review.current.impact" :key="item">{{ item }}</li>
                </ul>
              </section>
              <section>
                <h3>测试评估</h3>
                <p>
                  {{ review.current.test_assessment || "测试评估暂不可用。" }}
                </p>
              </section>
              <section>
                <h3>跳过文件</h3>
                <el-empty
                  v-if="review.current.skipped_files.length === 0"
                  description="暂无跳过文件"
                />
                <ul v-else class="plain-list mono">
                  <li v-for="file in review.current.skipped_files" :key="file">{{ file }}</li>
                </ul>
              </section>
              <section>
                <h3>Agent 指标</h3>
                <el-empty v-if="!latestLLMCall" description="暂无指标" />
                <dl v-else class="metrics-list">
                  <div>
                    <dt>Context 片段数</dt>
                    <dd>{{ latestLLMCall.context_chunks }}</dd>
                  </div>
                  <div>
                    <dt>保留片段数</dt>
                    <dd>{{ latestLLMCall.kept_chunks }}</dd>
                  </div>
                  <div>
                    <dt>规则 Findings</dt>
                    <dd>{{ latestLLMCall.rule_finding_count }}</dd>
                  </div>
                  <div>
                    <dt>跳过文件</dt>
                    <dd>{{ latestLLMCall.skipped_files }}</dd>
                  </div>
                  <div>
                    <dt>请求字节数</dt>
                    <dd>{{ latestLLMCall.request_bytes }}</dd>
                  </div>
                </dl>
              </section>
              <section>
                <h3>Agent 事件</h3>
                <el-empty v-if="events.length === 0" description="暂无事件" />
                <ul v-else class="plain-list event-list">
                  <li v-for="event in recentEvents" :key="event.id || event.created_at">
                    <span class="mono">{{ event.type }}</span>
                    <span>{{ event.message }}</span>
                  </li>
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

        <el-tab-pane label="上下文" name="context" lazy>
          <ContextPanel :review-id="reviewId" />
        </el-tab-pane>

        <el-tab-pane label="报告" name="report" lazy>
          <ReportPreview :review-id="reviewId" />
        </el-tab-pane>
      </el-tabs>
    </template>

    <el-empty v-else description="未找到 Review">
      <el-button type="primary" @click="router.push('/reviews')">返回 Review 列表</el-button>
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

import { getEvents } from "@/api/reviews";
import type { FindingFeedbackStatus, ReviewEvent } from "@/api/types";
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
const events = ref<ReviewEvent[]>([]);
const latestLLMCall = computed(() => {
  const calls = review.current?.llm_calls ?? [];
  return calls.length > 0 ? calls[calls.length - 1] : null;
});
const recentEvents = computed(() => events.value.slice(-8).reverse());

async function reload() {
  await review.fetchReview(reviewId.value);
  await loadEvents();
  if (!review.isTerminal) {
    review.startPolling(reviewId.value);
  }
}

async function loadEvents() {
  try {
    events.value = await getEvents(reviewId.value);
  } catch {
    events.value = [];
  }
}

async function setFeedback(id: string, status: FindingFeedbackStatus) {
  try {
    await review.setFindingFeedback(id, status);
    ElMessage.success("Feedback 已更新");
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : "更新 Feedback 失败");
  }
}

function formatDate(value: string) {
  if (!value) return "-";
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function formatDuration(value: number | null | undefined) {
  if (typeof value !== "number") return "-";
  return `${value} ms`;
}

onMounted(reload);
watch(reviewId, reload);
onBeforeUnmount(() => review.reset());
</script>
