<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2>Context</h2>
        <p>Repository snippets used during analysis.</p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadContexts">Refresh</el-button>
    </div>

    <el-alert
      v-if="error"
      class="section-alert"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />

    <el-table :data="contexts" row-key="id" border empty-text="No context chunks">
      <el-table-column prop="file" label="File" min-width="260" show-overflow-tooltip />
      <el-table-column prop="kind" label="Kind" width="130" />
      <el-table-column prop="tokens" label="Tokens" width="100" />
      <el-table-column label="Score" width="110">
        <template #default="{ row }">{{ formatScore(row.score) }}</template>
      </el-table-column>
      <el-table-column label="Content" min-width="300">
        <template #default="{ row }">
          <pre class="context-preview">{{ row.content }}</pre>
        </template>
      </el-table-column>
    </el-table>
  </section>
</template>

<script setup lang="ts">
import { Refresh } from "@element-plus/icons-vue";
import { onMounted, ref, watch } from "vue";

import { getContexts } from "@/api/reviews";
import type { ContextChunk } from "@/api/types";

const props = defineProps<{
  reviewId: string;
}>();

const contexts = ref<ContextChunk[]>([]);
const loading = ref(false);
const error = ref("");

async function loadContexts() {
  loading.value = true;
  error.value = "";
  try {
    contexts.value = await getContexts(props.reviewId);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load contexts";
  } finally {
    loading.value = false;
  }
}

onMounted(loadContexts);
watch(() => props.reviewId, loadContexts);

function formatScore(score: number | null | undefined) {
  return typeof score === "number" ? score.toFixed(2) : "-";
}
</script>
