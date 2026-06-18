<script setup lang="ts">
import { computed, ref } from "vue";
import { askAIAssistant, type AIAssistantResult } from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "SystemToolsAIAssistant" });

const question = ref("帮我总结最近系统日志");
const loading = ref(false);
const result = ref<AIAssistantResult>();

const examples = [
  "帮我总结最近系统日志",
  "分析最近异常",
  "生成用户操作报告",
  "今天谁登录失败最多？"
];

const rows = computed(() => result.value?.rows ?? []);
const columns = computed(() => {
  const first = rows.value[0];
  return first ? Object.keys(first) : [];
});

const ask = async (text = question.value) => {
  question.value = text;
  loading.value = true;
  try {
    const res = await askAIAssistant(question.value);
    result.value = res.result;
  } catch {
    message("AI 助手分析失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="ai-page">
    <section class="ai-hero">
      <div>
        <p class="eyebrow">Admin copilot</p>
        <h2>AI 助手</h2>
        <p>用自然语言总结日志、分析异常、生成用户操作报告。</p>
      </div>
    </section>

    <section class="ask-panel">
      <el-input
        v-model="question"
        type="textarea"
        :rows="3"
        resize="none"
        placeholder="例如：今天谁登录失败最多？"
      />
      <div class="ask-actions">
        <div class="example-list">
          <button v-for="item in examples" :key="item" type="button" @click="ask(item)">
            {{ item }}
          </button>
        </div>
        <el-button type="primary" :loading="loading" @click="ask()">开始分析</el-button>
      </div>
    </section>

    <section v-if="result" class="result-grid">
      <div class="result-card answer-card">
        <div class="card-title">分析结论</div>
        <p>{{ result.answer }}</p>
      </div>
      <div class="result-card">
        <div class="card-title">关键洞察</div>
        <ul class="insight-list">
          <li v-for="item in result.insights" :key="item">{{ item }}</li>
          <li v-if="!result.insights?.length">暂无洞察</li>
        </ul>
      </div>
      <div class="result-card full">
        <div class="card-title">数据明细</div>
        <el-table v-if="rows.length" :data="rows" stripe>
          <el-table-column
            v-for="column in columns"
            :key="column"
            :prop="column"
            :label="column"
            min-width="150"
            show-overflow-tooltip
          />
        </el-table>
        <el-empty v-else description="暂无明细数据" />
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.ai-page {
  padding: 24px;
  display: grid;
  gap: 16px;
}

.ai-hero,
.ask-panel,
.result-card {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.ai-hero {
  padding: 22px 24px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-violet);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.ai-hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 760;
}

.ai-hero p:last-child {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.ask-panel {
  padding: 18px;
}

.ask-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 12px;
}

.example-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.example-list button {
  height: 30px;
  padding: 0 10px;
  color: var(--app-primary-strong);
  cursor: pointer;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 999px;
}

.result-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.result-card {
  padding: 18px;
}

.result-card.full {
  grid-column: 1 / -1;
}

.card-title {
  margin-bottom: 10px;
  color: var(--app-text);
  font-weight: 760;
}

.answer-card p {
  margin: 0;
  color: var(--app-text-secondary);
  line-height: 1.7;
}

.insight-list {
  margin: 0;
  padding-left: 18px;
  color: var(--app-text-secondary);
  line-height: 1.8;
}

@media (max-width: 900px) {
  .ask-actions,
  .result-grid {
    align-items: flex-start;
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}
</style>
