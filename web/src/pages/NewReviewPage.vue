<template>
  <section class="page-stack">
    <div class="panel form-panel">
      <div class="panel-header">
        <div>
          <h2>Create Review</h2>
          <p>Submit a GitHub pull request URL for asynchronous analysis.</p>
        </div>
      </div>

      <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
        <el-form-item label="GitHub PR URL" prop="pr_url">
          <el-input
            v-model="form.pr_url"
            placeholder="https://github.com/org/repo/pull/123"
            clearable
          />
        </el-form-item>
        <el-form-item label="Reviewer ID" prop="reviewer_id">
          <el-input v-model="form.reviewer_id" placeholder="alice" clearable />
        </el-form-item>
        <el-form-item label="Model" prop="model">
          <el-select v-model="form.model" placeholder="Select model">
            <el-option
              v-for="model in modelChoices"
              :key="model"
              :label="model"
              :value="model"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Review Focus" prop="review_focus">
          <el-checkbox-group v-model="form.review_focus">
            <el-checkbox v-for="focus in focusOptions" :key="focus.value" :label="focus.value">
              {{ focus.label }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="Max Files" prop="max_files">
          <el-input-number v-model="form.max_files" :min="1" :max="300" :step="5" />
        </el-form-item>

        <div class="form-actions">
          <el-button @click="router.push('/reviews')">Cancel</el-button>
          <el-button type="primary" :loading="submitting" :icon="CaretRight" @click="submit">
            Start Analysis
          </el-button>
        </div>
      </el-form>
    </div>
  </section>
</template>

<script setup lang="ts">
import { CaretRight } from "@element-plus/icons-vue";
import { ElMessage, type FormInstance, type FormRules } from "element-plus";
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";

import { createReview } from "@/api/reviews";
import { useUserStore } from "@/stores/user";

interface ReviewForm {
  pr_url: string;
  reviewer_id: string;
  model: string;
  review_focus: string[];
  max_files: number;
}

const router = useRouter();
const user = useUserStore();
const formRef = ref<FormInstance>();
const submitting = ref(false);

const modelChoices = ["deepseek-chat", "gpt-4.1", "gpt-4.1-mini"];
const focusOptions = [
  { label: "Correctness", value: "correctness" },
  { label: "Security", value: "security" },
  { label: "Performance", value: "performance" },
  { label: "Compatibility", value: "compatibility" },
  { label: "Maintainability", value: "maintainability" },
  { label: "Test risk", value: "test risk" },
];

const form = reactive<ReviewForm>({
  pr_url: "",
  reviewer_id: user.reviewerId,
  model: "deepseek-chat",
  review_focus: ["correctness", "security", "test risk"],
  max_files: 50,
});

const githubPrPattern = /^https:\/\/github\.com\/[^/]+\/[^/]+\/pull\/\d+$/;

const rules: FormRules<ReviewForm> = {
  pr_url: [
    { required: true, message: "PR URL is required", trigger: "blur" },
    {
      validator: (_rule, value: string, callback) => {
        if (!githubPrPattern.test(value.trim())) {
          callback(new Error("Use https://github.com/{owner}/{repo}/pull/{number}"));
          return;
        }
        callback();
      },
      trigger: "blur",
    },
  ],
  reviewer_id: [{ required: true, message: "Reviewer ID is required", trigger: "blur" }],
  model: [{ required: true, message: "Model is required", trigger: "change" }],
  review_focus: [
    {
      type: "array",
      required: true,
      min: 1,
      message: "Select at least one focus",
      trigger: "change",
    },
  ],
  max_files: [{ required: true, message: "Max files is required", trigger: "change" }],
};

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;

  submitting.value = true;
  try {
    user.setReviewerId(form.reviewer_id);
    const result = await createReview({
      pr_url: form.pr_url.trim(),
      reviewer_id: form.reviewer_id.trim(),
      config: {
        model: form.model,
        review_focus: form.review_focus,
        max_files: form.max_files,
      },
    });
    ElMessage.success("Review session created");
    await router.push(`/reviews/${result.id}`);
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : "Failed to create review");
  } finally {
    submitting.value = false;
  }
}
</script>
