<template>
  <section class="page-stack">
    <div class="panel form-panel">
      <div class="panel-header">
        <div>
          <h2>新建 Review</h2>
          <p>提交 GitHub PR URL，创建异步分析任务。</p>
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
        <el-form-item label="模型" prop="model">
          <el-select v-model="form.model" placeholder="选择模型">
            <el-option
              v-for="model in modelChoices"
              :key="model"
              :label="model"
              :value="model"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="Review 关注点" prop="review_focus">
          <el-checkbox-group v-model="form.review_focus">
            <el-checkbox v-for="focus in focusOptions" :key="focus.value" :label="focus.value">
              {{ focus.label }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="最大文件数" prop="max_files">
          <el-input-number v-model="form.max_files" :min="1" :max="300" :step="5" />
        </el-form-item>

        <div class="form-actions">
          <el-button @click="router.push('/reviews')">取消</el-button>
          <el-button type="primary" :loading="submitting" :icon="CaretRight" @click="submit">
            开始分析
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
  { label: "正确性", value: "correctness" },
  { label: "安全性", value: "security" },
  { label: "性能", value: "performance" },
  { label: "兼容性", value: "compatibility" },
  { label: "可维护性", value: "maintainability" },
  { label: "测试风险", value: "test risk" },
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
    { required: true, message: "PR URL 不能为空", trigger: "blur" },
    {
      validator: (_rule, value: string, callback) => {
        if (!githubPrPattern.test(value.trim())) {
          callback(new Error("请使用 https://github.com/{owner}/{repo}/pull/{number}"));
          return;
        }
        callback();
      },
      trigger: "blur",
    },
  ],
  reviewer_id: [{ required: true, message: "Reviewer ID 不能为空", trigger: "blur" }],
  model: [{ required: true, message: "模型不能为空", trigger: "change" }],
  review_focus: [
    {
      type: "array",
      required: true,
      min: 1,
      message: "至少选择一个 Review 关注点",
      trigger: "change",
    },
  ],
  max_files: [{ required: true, message: "最大文件数不能为空", trigger: "change" }],
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
    ElMessage.success("Review 会话已创建");
    await router.push(`/reviews/${result.id}`);
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : "创建 Review 失败");
  } finally {
    submitting.value = false;
  }
}
</script>
