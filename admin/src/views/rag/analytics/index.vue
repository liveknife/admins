<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  getRAGAnalytics,
  getRAGQueryLogs,
  type RAGAnalytics,
  type RAGQueryLog
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGAnalytics" });

const loading = ref(false);
const analytics = ref<RAGAnalytics>();
const logs = ref<RAGQueryLog[]>([]);

const hitRatePercent = computed(() =>
  Math.round((analytics.value?.hit_rate ?? 0) * 100)
);

const percent = (value = 0) => `${Math.round(value * 100)}%`;
const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";

const loadData = async () => {
  loading.value = true;
  try {
    const [analyticsRes, logsRes] = await Promise.all([
      getRAGAnalytics(500),
      getRAGQueryLogs(100)
    ]);
    analytics.value = analyticsRes.analytics;
    logs.value = logsRes.logs ?? [];
  } catch {
    message("RAG 命中分析加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="analytics-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Hit analytics</p>
        <h2>知识库命中分析</h2>
        <p>观察命中率、低置信问题、无命中问题和来源分布，定位需要补文档或调权重的位置。</p>
      </div>
      <el-button @click="loadData">刷新</el-button>
    </section>

    <section class="metric-grid">
      <div class="metric-card">
        <span>总提问</span>
        <strong>{{ analytics?.query_count ?? 0 }}</strong>
        <small>{{ analytics?.hit_count ?? 0 }} 次命中</small>
      </div>
      <div class="metric-card">
        <span>命中率</span>
        <strong>{{ hitRatePercent }}%</strong>
        <el-progress :percentage="hitRatePercent" :show-text="false" />
      </div>
      <div class="metric-card">
        <span>平均耗时</span>
        <strong>{{ analytics?.avg_latency_ms ?? 0 }}ms</strong>
        <small>平均来源 {{ analytics?.avg_source_count ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>平均 Top score</span>
        <strong>{{ percent(analytics?.avg_top_score ?? 0) }}</strong>
        <small>越低越需要调阈值或补内容</small>
      </div>
    </section>

    <section class="layout">
      <div class="panel">
        <div class="panel-title">
          <h3>来源命中</h3>
          <span>按引用 chunk 聚合</span>
        </div>
        <el-table :data="analytics?.source_metrics ?? []" stripe>
          <el-table-column prop="name" label="来源" min-width="150" />
          <el-table-column prop="chunk_hits" label="命中 chunk" width="120" />
          <el-table-column label="平均分" width="100">
            <template #default="{ row }">{{ percent(row.avg_score) }}</template>
          </el-table-column>
          <el-table-column prop="avg_rank" label="平均排名" width="110" />
        </el-table>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>每日趋势</h3>
          <span>最近日志聚合</span>
        </div>
        <div class="daily-list">
          <div v-for="item in analytics?.daily_metrics ?? []" :key="item.date">
            <span>{{ item.date }}</span>
            <strong>{{ item.queries }}</strong>
            <el-progress :percentage="Math.round(item.hit_rate * 100)" />
          </div>
          <el-empty v-if="!analytics?.daily_metrics?.length" description="暂无趋势数据" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>低置信问题</h3>
          <span>命中但分数偏低</span>
        </div>
        <el-table :data="analytics?.low_confidence ?? []" stripe height="330">
          <el-table-column prop="question" label="问题" min-width="240" show-overflow-tooltip />
          <el-table-column label="Top score" width="110">
            <template #default="{ row }">{{ percent(row.top_score) }}</template>
          </el-table-column>
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ timeText(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>无命中问题</h3>
          <span>优先补充知识库</span>
        </div>
        <el-table :data="analytics?.no_hit_queries ?? []" stripe height="330">
          <el-table-column prop="question" label="问题" min-width="260" show-overflow-tooltip />
          <el-table-column label="耗时" width="100">
            <template #default="{ row }">{{ row.latency_ms }}ms</template>
          </el-table-column>
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ timeText(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel full">
        <div class="panel-title">
          <h3>最近查询日志</h3>
          <span>用于复盘具体问题</span>
        </div>
        <el-table :data="logs" stripe>
          <el-table-column prop="question" label="问题" min-width="260" show-overflow-tooltip />
          <el-table-column label="命中" width="90">
            <template #default="{ row }">
              <el-tag :type="row.matched ? 'success' : 'info'" effect="light">
                {{ row.matched ? "命中" : "未命中" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="source_count" label="来源" width="80" />
          <el-table-column label="Top score" width="110">
            <template #default="{ row }">{{ percent(row.top_score) }}</template>
          </el-table-column>
          <el-table-column label="耗时" width="100">
            <template #default="{ row }">{{ row.latency_ms }}ms</template>
          </el-table-column>
          <el-table-column label="时间" width="160">
            <template #default="{ row }">{{ timeText(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.analytics-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.page-head,
.metric-card,
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
.metric-card span,
.metric-card small,
.daily-list span {
  color: var(--app-text-secondary);
}

.page-head p {
  margin: 8px 0 0;
}

.metric-grid,
.layout {
  display: grid;
  gap: 16px;
}

.metric-grid {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.metric-card {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 18px;
}

.metric-card strong {
  color: var(--app-text);
  font-size: 26px;
}

.layout {
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.daily-list {
  display: grid;
  gap: 10px;
}

.daily-list > div {
  display: grid;
  grid-template-columns: 110px 42px minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  padding: 10px 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.daily-list strong {
  color: var(--app-text);
}

@media (max-width: 1100px) {
  .page-head,
  .panel-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .metric-grid,
  .layout {
    grid-template-columns: 1fr;
  }

  .daily-list > div {
    grid-template-columns: 1fr;
  }
}
</style>
