<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2>Context 上下文</h2>
        <p>分析过程中使用的代码上下文片段。</p>
      </div>
      <el-button :icon="Refresh" :loading="loading" @click="loadContexts">刷新</el-button>
    </div>

    <el-alert
      v-if="error"
      class="section-alert"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />

    <el-table :data="contexts" row-key="id" border empty-text="暂无 Context 片段">
      <el-table-column prop="file" label="文件" min-width="260" show-overflow-tooltip />
      <el-table-column prop="kind" label="类型" width="130" />
      <el-table-column prop="tokens" label="Tokens" width="100" />
      <el-table-column label="评分" width="110">
        <template #default="{ row }">{{ formatScore(row.score) }}</template>
      </el-table-column>
      <el-table-column label="内容" min-width="300">
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
    error.value = err instanceof Error ? err.message : "加载 Context 失败";
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
