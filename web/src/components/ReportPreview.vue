<template>
  <section class="panel">
    <div class="panel-header">
      <div>
        <h2>Markdown 报告</h2>
        <p>预览或导出生成的 Review 报告。</p>
      </div>
      <div class="toolbar">
        <el-button :icon="Refresh" :loading="loading" @click="loadReport">刷新</el-button>
        <el-button :icon="Download" type="primary" tag="a" :href="reportURL" target="_blank">
          导出
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
    <div v-if="markdown" class="markdown-preview" v-html="renderedMarkdown"></div>
    <el-empty v-else-if="!loading" description="暂无报告" />
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
const renderedMarkdown = computed(() => renderMarkdown(markdown.value));

async function loadReport() {
  loading.value = true;
  error.value = "";
  try {
    const response = await fetch(reportURL.value);
    if (!response.ok) {
      throw new Error(`报告请求失败：${response.status}`);
    }
    markdown.value = await response.text();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载报告失败";
  } finally {
    loading.value = false;
  }
}

onMounted(loadReport);
watch(() => props.reviewId, loadReport);

function renderMarkdown(source: string) {
  const lines = source.replace(/\r\n/g, "\n").split("\n");
  const blocks: string[] = [];
  let paragraph: string[] = [];
  let list: string[] = [];
  let code: string[] = [];
  let inCode = false;

  const flushParagraph = () => {
    if (paragraph.length === 0) return;
    blocks.push(`<p>${renderInline(paragraph.join(" "))}</p>`);
    paragraph = [];
  };

  const flushList = () => {
    if (list.length === 0) return;
    blocks.push(`<ul>${list.map((item) => `<li>${renderInline(item)}</li>`).join("")}</ul>`);
    list = [];
  };

  const flushCode = () => {
    blocks.push(`<pre><code>${escapeHtml(code.join("\n"))}</code></pre>`);
    code = [];
  };

  for (const line of lines) {
    if (line.trim().startsWith("```")) {
      if (inCode) {
        flushCode();
        inCode = false;
      } else {
        flushParagraph();
        flushList();
        inCode = true;
      }
      continue;
    }

    if (inCode) {
      code.push(line);
      continue;
    }

    const heading = /^(#{1,6})\s+(.+)$/.exec(line);
    if (heading) {
      flushParagraph();
      flushList();
      const level = heading[1].length;
      blocks.push(`<h${level}>${renderInline(heading[2])}</h${level}>`);
      continue;
    }

    const listItem = /^\s*[-*]\s+(.+)$/.exec(line);
    if (listItem) {
      flushParagraph();
      list.push(listItem[1]);
      continue;
    }

    if (line.trim() === "") {
      flushParagraph();
      flushList();
      continue;
    }

    flushList();
    paragraph.push(line.trim());
  }

  flushParagraph();
  flushList();
  if (inCode) flushCode();

  return blocks.join("");
}

function renderInline(source: string) {
  return escapeHtml(source)
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
}

function escapeHtml(source: string) {
  return source
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}
</script>
