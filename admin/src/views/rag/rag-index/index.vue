<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  getRAGIndexJobs,
  getRAGIndexStats,
  getRAGQueryLogs,
  rebuildRAGIndex,
  retryRAGIndexJob,
  runRAGEval,
  searchRAGDiagnostics,
  type KnowledgeSource,
  type RAGEvalRun,
  type RAGIndexJob,
  type RAGIndexStats,
  type RAGQueryLog
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGIndex" });

const loading = ref(false);
const rebuilding = ref(false);
const retryingId = ref<number>();
const diagnosing = ref(false);
const evalLoading = ref(false);
const stats = ref<RAGIndexStats>();
const jobs = ref<RAGIndexJob[]>([]);
const logs = ref<RAGQueryLog[]>([]);
const diagnosticQuestion = ref("这个项目使用了哪些技术栈？");
const includeInternal = ref(true);
const diagnosticSources = ref<KnowledgeSource[]>([]);
const evalRun = ref<RAGEvalRun>();

const hitRate = computed(() => {
  const total = stats.value?.query_count ?? 0;
  if (!total) return 0;
  return Math.round(((stats.value?.hit_count ?? 0) / total) * 100);
});

const sourceTypeLabels: Record<string, string> = {
  site_resource: "文章资源",
  site_project: "项目",
  site_tech_stack: "技术栈",
  site_timeline: "时间线",
  uploaded_document: "上传文档"
};

const sourceTypeText = (type: string) => sourceTypeLabels[type] ?? type;
const percent = (value = 0) => `${Math.round(value * 100)}%`;
const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";

const jobType = (status: string) => {
  if (status === "success") return "success";
  if (status === "failed") return "danger";
  if (status === "running" || status === "retrying") return "warning";
  return "info";
};

const parseSources = (value: string): KnowledgeSource[] => {
  try {
    const parsed = JSON.parse(value || "[]");
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
};

const loadData = async () => {
  loading.value = true;
  try {
    const [statsRes, jobsRes, logsRes] = await Promise.all([
      getRAGIndexStats(),
      getRAGIndexJobs(20),
      getRAGQueryLogs(50)
    ]);
    stats.value = statsRes.stats;
    jobs.value = jobsRes.jobs ?? [];
    logs.value = logsRes.logs ?? [];
  } catch {
    message("RAG 索引数据加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const submitRebuild = async () => {
  rebuilding.value = true;
  try {
    await rebuildRAGIndex();
    message("RAG 索引重建任务已提交", { type: "success" });
    await loadData();
  } catch {
    message("RAG 索引重建任务提交失败", { type: "error" });
  } finally {
    rebuilding.value = false;
  }
};

const retryJob = async (job: RAGIndexJob) => {
  retryingId.value = job.id;
  try {
    await retryRAGIndexJob(job.id);
    message("RAG 索引任务已重新提交", { type: "success" });
    await loadData();
  } catch {
    message("RAG 索引任务重试失败", { type: "error" });
  } finally {
    retryingId.value = undefined;
  }
};

const runDiagnostics = async () => {
  if (!diagnosticQuestion.value.trim()) {
    message("请输入诊断问题", { type: "warning" });
    return;
  }
  diagnosing.value = true;
  try {
    const res = await searchRAGDiagnostics({
      question: diagnosticQuestion.value,
      include_internal: includeInternal.value,
      top_k: stats.value?.top_k || 6
    });
    diagnosticSources.value = res.sources ?? [];
  } catch {
    message("检索诊断失败", { type: "error" });
  } finally {
    diagnosing.value = false;
  }
};

const runEval = async () => {
  evalLoading.value = true;
  try {
    const res = await runRAGEval(includeInternal.value);
    evalRun.value = res.run;
    message("RAG 评测完成", { type: "success" });
  } catch {
    message("RAG 评测失败", { type: "error" });
  } finally {
    evalLoading.value = false;
  }
};

onMounted(() => {
  loadData();
  runDiagnostics();
});
</script>

<template>
  <div class="rag-page" v-loading="loading">
    <section class="rag-head">
      <div>
        <p class="eyebrow">Knowledge operations</p>
        <h2>RAG 索引与检索质量</h2>
        <p>查看索引状态、混合检索分数、命中日志和固定评测集结果。</p>
      </div>
      <div class="head-actions">
        <el-button @click="loadData">
          <IconifyIconOnline icon="ri:refresh-line" />
          刷新
        </el-button>
        <el-button type="primary" :loading="rebuilding" @click="submitRebuild">
          <IconifyIconOnline icon="ri:loop-left-line" />
          异步重建索引
        </el-button>
      </div>
    </section>

    <section class="metric-grid">
      <div class="metric-card">
        <span>索引片段</span>
        <strong>{{ stats?.total_chunks ?? 0 }}</strong>
        <small>Top K {{ stats?.top_k ?? "-" }} · Rerank {{ stats?.rerank_top_n ?? "-" }}</small>
      </div>
      <div class="metric-card">
        <span>问答命中率</span>
        <strong>{{ hitRate }}%</strong>
        <small>{{ stats?.hit_count ?? 0 }} / {{ stats?.query_count ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>相似度阈值</span>
        <strong>{{ percent(stats?.min_score ?? 0) }}</strong>
        <el-progress :percentage="Math.round((stats?.min_score ?? 0) * 100)" :show-text="false" />
      </div>
      <div class="metric-card">
        <span>生成能力</span>
        <strong>{{ stats?.streaming_enabled ? "Streaming" : stats?.chat_enabled ? "Chat" : "Search" }}</strong>
        <small>{{ stats?.vector_backend ?? "json" }} · {{ stats?.pgvector_available ? "pgvector" : "portable" }}</small>
      </div>
    </section>

    <section class="content-grid">
      <div class="panel source-panel">
        <div class="panel-title">
          <h3>来源分布</h3>
          <span>按索引片段统计</span>
        </div>
        <div class="source-stats">
          <div v-for="(count, type) in stats?.by_source ?? {}" :key="type">
            <span>{{ sourceTypeText(type) }}</span>
            <strong>{{ count }}</strong>
          </div>
          <el-empty v-if="!Object.keys(stats?.by_source ?? {}).length" description="暂无索引片段" />
        </div>
      </div>

      <div class="panel weight-panel">
        <div class="panel-title">
          <h3>来源权重</h3>
          <span>用于 rerank 排序</span>
        </div>
        <div class="weight-list">
          <div v-for="(weight, type) in stats?.source_weights ?? {}" :key="type">
            <span>{{ sourceTypeText(type) }}</span>
            <el-progress :percentage="Math.min(Math.round(weight * 100), 100)" :format="() => `${weight}x`" />
          </div>
        </div>
      </div>

      <div class="panel diagnostic-panel">
        <div class="panel-title">
          <h3>检索诊断</h3>
          <span>BM25 + 向量 + 来源权重</span>
        </div>
        <div class="diagnostic-form">
          <el-input
            v-model="diagnosticQuestion"
            placeholder="输入一个真实业务问题"
            clearable
            @keyup.enter="runDiagnostics"
          />
          <el-switch
            v-model="includeInternal"
            active-text="含 Internal"
            inactive-text="仅 Public"
            inline-prompt
          />
          <el-button type="primary" :loading="diagnosing" @click="runDiagnostics">诊断</el-button>
        </div>
        <div class="diagnostic-results">
          <article v-for="item in diagnosticSources" :key="`${item.chunk_id}-${item.title}`" class="source-card">
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
          <el-empty v-if="!diagnosticSources.length" description="暂无诊断结果" />
        </div>
      </div>

      <div class="panel jobs-panel">
        <div class="panel-title">
          <h3>最近任务</h3>
          <span>重建与失败重试</span>
        </div>
        <el-table :data="jobs" stripe height="338">
          <el-table-column label="任务" width="86">
            <template #default="{ row }">#{{ row.id }}</template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="jobType(row.status)" effect="light">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="重试" width="86">
            <template #default="{ row }">{{ row.retry_count }} / {{ row.max_retries }}</template>
          </el-table-column>
          <el-table-column label="更新" width="150">
            <template #default="{ row }">{{ timeText(row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="错误" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">{{ row.error_message || "-" }}</template>
          </el-table-column>
          <el-table-column label="操作" width="96" fixed="right">
            <template #default="{ row }">
              <el-button
                size="small"
                :disabled="row.status !== 'failed'"
                :loading="retryingId === row.id"
                @click="retryJob(row)"
              >
                重试
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel full">
        <el-tabs>
          <el-tab-pane label="命中日志">
            <el-table :data="logs" stripe>
              <el-table-column prop="question" label="问题" min-width="240" show-overflow-tooltip />
              <el-table-column label="命中" width="90">
                <template #default="{ row }">
                  <el-tag :type="row.matched ? 'success' : 'info'" effect="light">
                    {{ row.matched ? "命中" : "未命中" }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="source_count" label="来源" width="80" />
              <el-table-column label="最高分" width="96">
                <template #default="{ row }">{{ percent(row.top_score) }}</template>
              </el-table-column>
              <el-table-column label="耗时" width="96">
                <template #default="{ row }">{{ row.latency_ms }} ms</template>
              </el-table-column>
              <el-table-column label="模型" width="90">
                <template #default="{ row }">{{ row.used_chat_model ? "Chat" : "检索" }}</template>
              </el-table-column>
              <el-table-column label="时间" width="160">
                <template #default="{ row }">{{ timeText(row.created_at) }}</template>
              </el-table-column>
              <el-table-column type="expand">
                <template #default="{ row }">
                  <div class="log-detail">
                    <p>{{ row.answer || "没有生成答案" }}</p>
                    <div class="source-list">
                      <article
                        v-for="item in parseSources(row.source_json)"
                        :key="`${item.chunk_id}-${item.title}`"
                      >
                        <strong>[{{ item.citation_id }}] {{ item.title }}</strong>
                        <span>{{ sourceTypeText(item.source_type) }} #{{ item.source_id }} · {{ percent(item.score) }}</span>
                        <em v-if="item.highlighted_text" v-html="item.highlighted_text" />
                      </article>
                      <el-empty v-if="!parseSources(row.source_json).length" description="暂无来源" />
                    </div>
                  </div>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane label="评测集">
            <div class="eval-head">
              <div>
                <strong>{{ evalRun ? `${evalRun.recall_hits}/${evalRun.total}` : "未运行" }}</strong>
                <span>召回命中 · 平均质量 {{ percent(evalRun?.average_quality ?? 0) }}</span>
              </div>
              <el-button type="primary" :loading="evalLoading" @click="runEval">运行固定评测</el-button>
            </div>
            <el-table :data="evalRun?.results ?? []" stripe>
              <el-table-column label="用例" min-width="240">
                <template #default="{ row }">
                  <strong>{{ row.case.id }}</strong>
                  <p>{{ row.case.question }}</p>
                </template>
              </el-table-column>
              <el-table-column label="召回" width="90">
                <template #default="{ row }">
                  <el-tag :type="row.recall_hit ? 'success' : 'danger'" effect="light">
                    {{ row.recall_hit ? "命中" : "偏离" }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="质量" width="120">
                <template #default="{ row }">{{ percent(row.answer_quality) }}</template>
              </el-table-column>
              <el-table-column label="最高分" width="120">
                <template #default="{ row }">{{ percent(row.top_score) }}</template>
              </el-table-column>
              <el-table-column label="来源" min-width="220">
                <template #default="{ row }">
                  <span>{{ row.sources.map(item => `[${item.citation_id}] ${item.title}`).join(" / ") || "-" }}</span>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.rag-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.rag-head,
.metric-card,
.panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.rag-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-primary);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.rag-head h2,
.panel-title h3 {
  margin: 0;
  color: var(--app-text);
}

.rag-head p {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.head-actions,
.diagnostic-form,
.eval-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 18px;
}

.metric-card span,
.metric-card small,
.panel-title span,
.score-grid span,
.source-card p,
.log-detail p,
.source-list span,
.source-list em,
.eval-head span {
  color: var(--app-text-secondary);
}

.metric-card strong {
  color: var(--app-text);
  font-size: 26px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
  gap: 16px;
}

.panel {
  min-width: 0;
  padding: 18px;
}

.panel.full {
  grid-column: 1 / -1;
}

.diagnostic-panel,
.jobs-panel {
  min-height: 420px;
}

.panel-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.source-stats,
.weight-list,
.diagnostic-results,
.source-list {
  display: grid;
  gap: 10px;
}

.source-stats > div,
.weight-list > div,
.source-card,
.source-list article {
  display: grid;
  gap: 8px;
  padding: 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.source-stats > div {
  grid-template-columns: 1fr auto;
}

.source-stats strong,
.source-card strong,
.source-list strong,
.eval-head strong {
  color: var(--app-text);
}

.diagnostic-form {
  margin-bottom: 14px;
}

.source-card header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.source-card p,
.log-detail p,
.source-list em {
  margin: 0;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.score-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
}

.log-detail {
  display: grid;
  gap: 12px;
  padding: 6px 28px 18px;
}

.source-list :deep(mark),
.source-card :deep(mark) {
  padding: 0 3px;
  color: var(--app-text);
  background: rgb(245 158 11 / 22%);
  border-radius: 4px;
}

.eval-head {
  justify-content: space-between;
  margin-bottom: 12px;
}

.eval-head > div {
  display: grid;
  gap: 4px;
}

@media (max-width: 1100px) {
  .rag-head,
  .head-actions,
  .panel-title,
  .diagnostic-form,
  .eval-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .metric-grid,
  .content-grid,
  .score-grid {
    grid-template-columns: 1fr;
  }
}
</style>
