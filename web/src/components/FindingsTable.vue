<template>
  <el-table :data="findings" row-key="id" border empty-text="暂无 Findings">
    <el-table-column type="expand">
      <template #default="{ row }">
        <div class="finding-detail">
          <div>
            <div class="field-label">依据</div>
            <p>{{ row.evidence || "暂无依据。" }}</p>
          </div>
          <div>
            <div class="field-label">建议</div>
            <p>{{ row.suggestion || "暂无建议。" }}</p>
          </div>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="风险级别" width="110">
      <template #default="{ row }">
        <el-tag :type="severityType(row.severity)" effect="dark">
          {{ row.severity }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="title" label="Finding 标题" min-width="260" show-overflow-tooltip />
    <el-table-column prop="category" label="分类" width="150" />
    <el-table-column label="置信度" width="120">
      <template #default="{ row }">{{ formatConfidence(row.confidence) }}</template>
    </el-table-column>
    <el-table-column label="位置" min-width="220" show-overflow-tooltip>
      <template #default="{ row }">
        <span class="mono">{{ formatLocation(row.file, row.line) }}</span>
      </template>
    </el-table-column>
    <el-table-column label="人工确认" width="125">
      <template #default="{ row }">
        <el-tag v-if="row.needs_human_check" type="warning">需要</el-tag>
        <el-tag v-else type="info">否</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="反馈" width="170" fixed="right">
      <template #default="{ row }">
        <el-select
          :model-value="row.feedback_status ?? ''"
          placeholder="设置状态"
          size="small"
          @change="(status: string) => emitFeedback(row.id, status)"
        >
          <el-option label="有用" value="useful" />
          <el-option label="误报" value="false_positive" />
          <el-option label="已修复" value="fixed" />
          <el-option label="忽略" value="ignored" />
        </el-select>
      </template>
    </el-table-column>
  </el-table>
</template>

<script setup lang="ts">
import type { Finding, FindingFeedbackStatus, FindingSeverity } from "@/api/types";

defineProps<{
  findings: Finding[];
}>();

const emit = defineEmits<{
  feedback: [id: string, status: FindingFeedbackStatus];
}>();

function severityType(severity: FindingSeverity) {
  if (severity === "high") return "danger";
  if (severity === "medium") return "warning";
  return "info";
}

function formatConfidence(value: number) {
  return `${Math.round((value || 0) * 100)}%`;
}

function formatLocation(file: string, line: number) {
  return line > 0 ? `${file}:${line}` : file || "-";
}

function emitFeedback(id: string, value: string | number | boolean | object) {
  emit("feedback", id, value as FindingFeedbackStatus);
}
</script>
