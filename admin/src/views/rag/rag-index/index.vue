<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  getRAGIndexJobs,
  getRAGIndexStats,
  getRAGQueryLogs,
  rebuildRAGIndex,
  retryRAGIndexJob,
  type KnowledgeSource,
  type RAGIndexJob,
  type RAGIndexStats,
  type RAGQueryLog
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGIndex" });

const loading = ref(false);
const rebuilding = ref(false);
const retryingId = ref<number>();
const stats = ref<RAGIndexStats>();
const jobs = ref<RAGIndexJob[]>([]);
const logs = ref<RAGQueryLog[]>([]);

const hitRate = computed(() => {
  const total = stats.value?.query_count ?? 0;
  if (!total) return "0%";
  return `${Math.round(((stats.value?.hit_count ?? 0) / total) * 100)}%`;
});

const sourceTypeText = (type: string) => {
  const map: Record<string, string> = {
    site_resource: "资源",
    site_project: "项目",
    site_tech_stack: "技术栈",
    site_timeline: "时间线",
    uploaded_document: "上传文档"
  };
  return map[type] ?? type;
};

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

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";

const loadData = async () => {
  loading.value = true;
  try {
    const [statsRes, jobsRes, logsRes] = await Promise.all([
      getRAGIndexStats(),
      getRAGIndexJobs(20),
      getRAGQueryLogs(30)
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

onMounted(loadData);
</script>

<template>
  <div class="rag-page" v-loading="loading">
    <section class="rag-head">
      <div>
        <p class="eyebrow">Knowledge operations</p>
        <h2>RAG 索引管理</h2>
        <p>查看知识库索引、异步重建任务、失败重试和问答命中情况。</p>
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
        <small>Top K {{ stats?.top_k ?? "-" }}</small>
      </div>
      <div class="metric-card">
        <span>问答命中率</span>
        <strong>{{ hitRate }}</strong>
        <small>{{ stats?.hit_count ?? 0 }} / {{ stats?.query_count ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>平均耗时</span>
        <strong>{{ Math.round(stats?.average_latency_ms ?? 0) }}</strong>
        <small>ms</small>
      </div>
      <div class="metric-card">
        <span>向量后端</span>
        <strong>{{ stats?.vector_backend ?? "json" }}</strong>
        <small>{{ stats?.pgvector_available ? "pgvector available" : "portable JSON" }}</small>
      </div>
    </section>

    <section class="content-grid">
      <div class="panel">
        <div class="panel-title">
          <h3>索引分布</h3>
          <span>按来源类型统计</span>
        </div>
        <div class="source-stats">
          <div v-for="(count, type) in stats?.by_source ?? {}" :key="type">
            <span>{{ sourceTypeText(type) }}</span>
            <strong>{{ count }}</strong>
          </div>
          <el-empty v-if="!Object.keys(stats?.by_source ?? {}).length" description="暂无索引片段" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>最近任务</h3>
          <span>异步重建与失败重试</span>
        </div>
        <el-table :data="jobs" stripe>
          <el-table-column label="任务" width="92">
            <template #default="{ row }">#{{ row.id }}</template>
          </el-table-column>
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="jobType(row.status)" effect="light">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="重试" width="90">
            <template #default="{ row }">{{ row.retry_count }} / {{ row.max_retries }}</template>
          </el-table-column>
          <el-table-column label="更新时间" width="180">
            <template #default="{ row }">{{ timeText(row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="错误" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.error_message || "-" }}</template>
          </el-table-column>
          <el-table-column label="操作" width="110" fixed="right">
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
        <div class="panel-title">
          <h3>问答日志</h3>
          <span>命中、来源数、耗时和来源预览</span>
        </div>
        <el-table :data="logs" stripe>
          <el-table-column prop="question" label="问题" min-width="220" show-overflow-tooltip />
          <el-table-column label="命中" width="90">
            <template #default="{ row }">
              <el-tag :type="row.matched ? 'success' : 'info'" effect="light">
                {{ row.matched ? "命中" : "未命中" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="source_count" label="来源" width="90" />
          <el-table-column label="最高分" width="100">
            <template #default="{ row }">{{ Math.round(row.top_score * 100) }}%</template>
          </el-table-column>
          <el-table-column label="耗时" width="100">
            <template #default="{ row }">{{ row.latency_ms }} ms</template>
          </el-table-column>
          <el-table-column label="模型" width="90">
            <template #default="{ row }">{{ row.used_chat_model ? "Chat" : "检索" }}</template>
          </el-table-column>
          <el-table-column label="时间" width="180">
            <template #default="{ row }">{{ timeText(row.created_at) }}</template>
          </el-table-column>
          <el-table-column type="expand">
            <template #default="{ row }">
              <div class="log-detail">
                <p>{{ row.answer || "没有生成答案" }}</p>
                <div class="source-list">
                  <a
                    v-for="item in parseSources(row.source_json)"
                    :key="`${item.source_type}-${item.source_id}-${item.title}`"
                    :href="item.url || '#'"
                    :target="item.url?.startsWith('http') ? '_blank' : undefined"
                    rel="noreferrer"
                  >
                    <strong>{{ item.title }}</strong>
                    <span>{{ sourceTypeText(item.source_type) }} · {{ Math.round(item.score * 100) }}%</span>
                    <em v-if="item.highlighted_text" v-html="item.highlighted_text" />
                  </a>
                  <el-empty v-if="!parseSources(row.source_json).length" description="暂无来源" />
                </div>
              </div>
            </template>
          </el-table-column>
        </el-table>
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

.head-actions {
  display: flex;
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
  padding: 18px;
}

.metric-card span,
.metric-card small,
.panel-title span {
  color: var(--app-text-secondary);
}

.metric-card strong {
  color: var(--app-text);
  font-size: 26px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(280px, 380px) minmax(0, 1fr);
  gap: 16px;
}

.panel {
  min-width: 0;
  padding: 18px;
}

.panel.full {
  grid-column: 1 / -1;
}

.panel-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.source-stats {
  display: grid;
  gap: 10px;
}

.source-stats > div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.log-detail {
  display: grid;
  gap: 12px;
  padding: 6px 28px 18px;
}

.log-detail p {
  margin: 0;
  color: var(--app-text-secondary);
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.source-list {
  display: grid;
  gap: 10px;
}

.source-list a {
  display: grid;
  gap: 4px;
  padding: 12px;
  color: inherit;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.source-list strong {
  color: var(--app-text);
}

.source-list span,
.source-list em {
  color: var(--app-text-secondary);
  font-style: normal;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.source-list :deep(mark) {
  padding: 0 3px;
  color: var(--app-text);
  background: rgb(245 158 11 / 22%);
  border-radius: 4px;
}

@media (max-width: 1100px) {
  .rag-head,
  .head-actions,
  .panel-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .metric-grid,
  .content-grid {
    grid-template-columns: 1fr;
  }
}
</style>
