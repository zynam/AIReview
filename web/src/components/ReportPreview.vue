<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2>Markdown Report</h2>
        <p>Preview or export the generated review report.</p>
      </div>
      <div class="toolbar">
        <el-button :icon="Refresh" :loading="loading" @click="loadReport">Refresh</el-button>
        <el-button :icon="Download" type="primary" tag="a" :href="reportURL" target="_blank">
          Export
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="error"
      class="section-alert"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />
    <pre v-if="markdown" class="report-preview">{{ markdown }}</pre>
    <el-empty v-else-if="!loading" description="No report available" />
  </section>
</template>

<script setup lang="ts">
import { Download, Refresh } from "@element-plus/icons-vue";
import { computed, onMounted, ref, watch } from "vue";

import { getReportURL } from "@/api/reviews";

const props = defineProps<{
  reviewId: string;
}>();

const markdown = ref("");
const loading = ref(false);
const error = ref("");
const reportURL = computed(() => getReportURL(props.reviewId));

async function loadReport() {
  loading.value = true;
  error.value = "";
  try {
    const response = await fetch(reportURL.value);
    if (!response.ok) {
      throw new Error(`Report request failed: ${response.status}`);
    }
    markdown.value = await response.text();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load report";
  } finally {
    loading.value = false;
  }
}

onMounted(loadReport);
watch(() => props.reviewId, loadReport);
</script>
