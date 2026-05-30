<template>
  <el-tag :type="tagType" effect="light" round>{{ label }}</el-tag>
</template>

<script setup lang="ts">
import { computed } from "vue";

import type { ReviewStatus } from "@/api/types";

const props = defineProps<{
  status: ReviewStatus | "";
}>();

const label = computed(() => props.status || "unknown");

const tagType = computed(() => {
  switch (props.status) {
    case "completed":
      return "success";
    case "failed":
      return "danger";
    case "cancelled":
      return "info";
    case "analyzing":
    case "building_context":
    case "fetching_pr":
      return "warning";
    default:
      return "info";
  }
});
</script>
