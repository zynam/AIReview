<template>
  <el-table :data="findings" row-key="id" border empty-text="No findings">
    <el-table-column type="expand">
      <template #default="{ row }">
        <div class="finding-detail">
          <div>
            <div class="field-label">Evidence</div>
            <p>{{ row.evidence || "No evidence provided." }}</p>
          </div>
          <div>
            <div class="field-label">Suggestion</div>
            <p>{{ row.suggestion || "No suggestion provided." }}</p>
          </div>
        </div>
      </template>
    </el-table-column>
    <el-table-column label="Severity" width="110">
      <template #default="{ row }">
        <el-tag :type="severityType(row.severity)" effect="dark">
          {{ row.severity }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="title" label="Finding" min-width="260" show-overflow-tooltip />
    <el-table-column prop="category" label="Category" width="150" />
    <el-table-column label="Confidence" width="120">
      <template #default="{ row }">{{ formatConfidence(row.confidence) }}</template>
    </el-table-column>
    <el-table-column label="Location" min-width="220" show-overflow-tooltip>
      <template #default="{ row }">
        <span class="mono">{{ formatLocation(row.file, row.line) }}</span>
      </template>
    </el-table-column>
    <el-table-column label="Human Check" width="125">
      <template #default="{ row }">
        <el-tag v-if="row.needs_human_check" type="warning">required</el-tag>
        <el-tag v-else type="info">no</el-tag>
      </template>
    </el-table-column>
    <el-table-column label="Feedback" width="170" fixed="right">
      <template #default="{ row }">
        <el-select
          :model-value="row.feedback_status ?? ''"
          placeholder="Set status"
          size="small"
          @change="(status: string) => emitFeedback(row.id, status)"
        >
          <el-option label="Useful" value="useful" />
          <el-option label="False positive" value="false_positive" />
          <el-option label="Fixed" value="fixed" />
          <el-option label="Ignored" value="ignored" />
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
