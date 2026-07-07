<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import {
  getRAGConfig,
  saveRAGConfig,
  searchRAGDiagnostics,
  type KnowledgeSource,
  type RAGConfig
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGTuning" });

const loading = ref(false);
const saving = ref(false);
const diagnosing = ref(false);
const question = ref("这个项目使用了哪些前后端技术？");
const includeInternal = ref(true);
const sources = ref<KnowledgeSource[]>([]);

const sourceLabels: Record<string, string> = {
  site_resource: "文章资源",
  site_project: "项目",
  site_tech_stack: "技术栈",
  site_timeline: "时间线",
  uploaded_document: "上传文档"
};

const form = reactive<RAGConfig>({
  top_k: 6,
  min_score: 0.08,
  rerank_top_n: 36,
  vector_weight: 0.58,
  bm25_weight: 0.32,
  keyword_weight: 0.1,
  title_boost: 0.04,
  source_weights: {
    site_resource: 1,
    site_project: 1,
    site_tech_stack: 1,
    site_timeline: 1,
    uploaded_document: 1
  }
});

const sourceWeightRows = computed(() =>
  Object.keys(form.source_weights).map(key => ({
    key,
    label: sourceLabels[key] ?? key,
    value: form.source_weights[key]
  }))
);

const percent = (value = 0) => `${Math.round(value * 100)}%`;

const applyConfig = (config: RAGConfig) => {
  Object.assign(form, config);
  form.source_weights = {
    site_resource: 1,
    site_project: 1,
    site_tech_stack: 1,
    site_timeline: 1,
    uploaded_document: 1,
    ...(config.source_weights ?? {})
  };
};

const loadConfig = async () => {
  loading.value = true;
  try {
    const res = await getRAGConfig();
    applyConfig(res.config);
  } catch {
    message("RAG 调参配置加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const submit = async () => {
  saving.value = true;
  try {
    const res = await saveRAGConfig({ ...form });
    applyConfig(res.config);
    message("RAG 调参配置已保存", { type: "success" });
    await diagnose();
  } catch {
    message("RAG 调参配置保存失败", { type: "error" });
  } finally {
    saving.value = false;
  }
};

const diagnose = async () => {
  if (!question.value.trim()) {
    message("请输入诊断问题", { type: "warning" });
    return;
  }
  diagnosing.value = true;
  try {
    const res = await searchRAGDiagnostics({
      question: question.value,
      include_internal: includeInternal.value,
      top_k: form.top_k
    });
    sources.value = res.sources ?? [];
  } catch {
    message("检索诊断失败", { type: "error" });
  } finally {
    diagnosing.value = false;
  }
};

onMounted(async () => {
  await loadConfig();
  diagnose();
});
</script>

<template>
  <div class="rag-tool-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Retrieval tuning</p>
        <h2>RAG 调参中心</h2>
        <p>调整召回数量、相似度阈值、混合检索权重和来源权重，右侧直接查看命中变化。</p>
      </div>
      <div class="head-actions">
        <el-button @click="loadConfig">刷新</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存配置</el-button>
      </div>
    </section>

    <section class="layout">
      <div class="panel config-panel">
        <div class="panel-title">
          <h3>检索参数</h3>
          <span>保存后立即影响官网和后台 RAG 检索</span>
        </div>

        <div class="control-grid">
          <label>
            <span>Top K</span>
            <el-input-number v-model="form.top_k" :min="1" :max="20" />
          </label>
          <label>
            <span>Rerank Top N</span>
            <el-input-number v-model="form.rerank_top_n" :min="form.top_k" :max="200" />
          </label>
        </div>

        <div class="slider-block">
          <div>
            <strong>相似度阈值</strong>
            <span>{{ percent(form.min_score) }}</span>
          </div>
          <el-slider v-model="form.min_score" :min="0" :max="1" :step="0.01" />
        </div>

        <div class="weight-grid">
          <div class="slider-block">
            <div>
              <strong>向量权重</strong>
              <span>{{ form.vector_weight.toFixed(2) }}</span>
            </div>
            <el-slider v-model="form.vector_weight" :min="0" :max="1" :step="0.01" />
          </div>
          <div class="slider-block">
            <div>
              <strong>BM25 权重</strong>
              <span>{{ form.bm25_weight.toFixed(2) }}</span>
            </div>
            <el-slider v-model="form.bm25_weight" :min="0" :max="1" :step="0.01" />
          </div>
          <div class="slider-block">
            <div>
              <strong>关键词权重</strong>
              <span>{{ form.keyword_weight.toFixed(2) }}</span>
            </div>
            <el-slider v-model="form.keyword_weight" :min="0" :max="1" :step="0.01" />
          </div>
          <div class="slider-block">
            <div>
              <strong>标题加权</strong>
              <span>{{ form.title_boost.toFixed(2) }}</span>
            </div>
            <el-slider v-model="form.title_boost" :min="0" :max="0.5" :step="0.01" />
          </div>
        </div>

        <div class="panel-title compact">
          <h3>来源权重</h3>
          <span>高权重来源会在 rerank 中靠前</span>
        </div>
        <div class="source-weight-list">
          <div v-for="item in sourceWeightRows" :key="item.key">
            <span>{{ item.label }}</span>
            <el-input-number
              v-model="form.source_weights[item.key]"
              :min="0.1"
              :max="3"
              :step="0.1"
              controls-position="right"
            />
          </div>
        </div>
      </div>

      <div class="panel diagnostic-panel">
        <div class="panel-title">
          <h3>即时诊断</h3>
          <span>查看每个 chunk 的分数构成</span>
        </div>
        <div class="diagnostic-form">
          <el-input v-model="question" clearable placeholder="输入一个业务问题" @keyup.enter="diagnose" />
          <el-switch
            v-model="includeInternal"
            active-text="含 Internal"
            inactive-text="仅 Public"
            inline-prompt
          />
          <el-button type="primary" :loading="diagnosing" @click="diagnose">诊断</el-button>
        </div>

        <div class="result-list">
          <article v-for="item in sources" :key="`${item.chunk_id}-${item.title}`">
            <header>
              <strong>[{{ item.citation_id }}] {{ item.title }}</strong>
              <el-tag size="small" effect="light">{{ item.visibility }}</el-tag>
            </header>
            <p v-if="item.highlighted_text" v-html="item.highlighted_text" />
            <div class="score-grid">
              <span>最终 {{ percent(item.score) }}</span>
              <span>向量 {{ percent(item.vector_score) }}</span>
              <span>BM25 {{ percent(item.bm25_score) }}</span>
              <span>关键词 {{ percent(item.keyword_score) }}</span>
              <span>权重 {{ item.source_weight || 1 }}x</span>
            </div>
          </article>
          <el-empty v-if="!sources.length" description="暂无诊断结果" />
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.rag-tool-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.page-head,
.panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-primary);
  font-family: Menlo, Consolas, monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.page-head h2,
.panel-title h3 {
  margin: 0;
  color: var(--app-text);
}

.page-head p,
.panel-title span,
.slider-block span,
.result-list p,
.score-grid span {
  color: var(--app-text-secondary);
}

.page-head p {
  margin: 8px 0 0;
}

.head-actions,
.diagnostic-form {
  display: flex;
  align-items: center;
  gap: 10px;
}

.layout {
  display: grid;
  grid-template-columns: minmax(340px, 480px) minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}

.panel {
  min-width: 0;
  padding: 18px;
}

.panel-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.panel-title.compact {
  margin-top: 20px;
}

.control-grid,
.weight-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.control-grid label,
.slider-block,
.source-weight-list > div,
.result-list article {
  display: grid;
  gap: 8px;
  padding: 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.control-grid label > span,
.slider-block strong,
.source-weight-list span,
.result-list strong {
  color: var(--app-text);
}

.slider-block {
  margin-top: 14px;
}

.slider-block > div,
.source-weight-list > div,
.result-list header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.source-weight-list,
.result-list {
  display: grid;
  gap: 10px;
}

.diagnostic-form {
  margin-bottom: 14px;
}

.result-list p {
  margin: 0;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.result-list :deep(mark) {
  padding: 0 3px;
  color: var(--app-text);
  background: rgb(245 158 11 / 22%);
  border-radius: 4px;
}

.score-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
  font-family: Menlo, Consolas, monospace;
  font-size: 12px;
}

@media (max-width: 1100px) {
  .page-head,
  .head-actions,
  .diagnostic-form,
  .panel-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .layout,
  .control-grid,
  .weight-grid,
  .score-grid {
    grid-template-columns: 1fr;
  }
}
</style>
