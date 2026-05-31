<template>
  <div class="progress-block">
    <el-steps :active="activeStep" finish-status="success" process-status="process">
      <el-step title="Created" />
      <el-step title="拉取 PR" />
      <el-step title="构建 Context" />
      <el-step title="分析" />
      <el-step title="完成" />
    </el-steps>
    <el-alert
      v-if="status === 'failed' && error"
      class="progress-alert"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

import type { ReviewStatus } from "@/api/types";

const props = defineProps<{
  status: ReviewStatus;
  error?: string;
}>();

const activeStep = computed(() => {
  switch (props.status) {
    case "created":
      return 1;
    case "fetching_pr":
      return 2;
    case "building_context":
      return 3;
    case "analyzing":
      return 4;
    case "completed":
    case "failed":
    case "cancelled":
      return 5;
    default:
      return 0;
  }
});
</script>
