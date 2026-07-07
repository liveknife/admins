<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  getAIModelCallLogs,
  getAIModelCallStats,
  type AIModelCallLog,
  type AIModelCallStats
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGAIModelCallLogs" });

const loading = ref(false);
const stats = ref<AIModelCallStats>();
const logs = ref<AIModelCallLog[]>([]);
const total = ref(0);
const detail = ref<AIModelCallLog>();
const query = reactive({
  page: 1,
  page_size: 15,
  provider: "",
  operation: "",
  status: ""
});

const successRate = computed(() => {
  const totalCalls = stats.value?.total_calls ?? 0;
  if (!totalCalls) return 0;
  return Math.round(((stats.value?.success_calls ?? 0) / totalCalls) * 100);
});

const loadData = async () => {
  loading.value = true;
  try {
    const [statsRes, logsRes] = await Promise.all([
      getAIModelCallStats(),
      getAIModelCallLogs(query)
    ]);
    stats.value = statsRes.stats;
    logs.value = logsRes.logs ?? [];
    total.value = logsRes.total ?? 0;
  } catch {
    message("AI 调用日志加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const search = () => {
  query.page = 1;
  loadData();
};

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";
</script>

<template>
  <div class="logs-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Model telemetry</p>
        <h2>AI 模型调用日志</h2>
        <p>记录 chat、stream、embedding 的模型、耗时、状态、近似 token 和错误信息。</p>
      </div>
      <el-button @click="loadData">刷新</el-button>
    </section>

    <section class="metric-grid">
      <div class="metric-card">
        <span>总调用</span>
        <strong>{{ stats?.total_calls ?? 0 }}</strong>
        <small>今日 {{ stats?.today_calls ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>成功率</span>
        <strong>{{ successRate }}%</strong>
        <el-progress :percentage="successRate" :show-text="false" />
      </div>
      <div class="metric-card">
        <span>平均耗时</span>
        <strong>{{ stats?.avg_latency_ms ?? 0 }}ms</strong>
        <small>失败 {{ stats?.error_calls ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>近似 token</span>
        <strong>{{ stats?.total_tokens ?? 0 }}</strong>
        <small>按字符估算</small>
      </div>
    </section>

    <section class="layout">
      <div class="panel side">
        <div class="panel-title">
          <h3>模型分布</h3>
          <span>Top 8</span>
        </div>
        <div class="model-list">
          <div v-for="item in stats?.model_stats ?? []" :key="`${item.provider}-${item.model}`">
            <div>
              <strong>{{ item.model || "未配置模型" }}</strong>
              <span>{{ item.provider }} · {{ item.avg_ms }}ms</span>
            </div>
            <b>{{ item.calls }}</b>
          </div>
          <el-empty v-if="!stats?.model_stats?.length" description="暂无调用数据" />
        </div>
      </div>

      <div class="panel main">
        <div class="toolbar">
          <el-select v-model="query.operation" placeholder="调用类型" clearable>
            <el-option label="chat" value="chat" />
            <el-option label="stream" value="stream" />
            <el-option label="embed" value="embed" />
          </el-select>
          <el-select v-model="query.status" placeholder="状态" clearable>
            <el-option label="success" value="success" />
            <el-option label="error" value="error" />
          </el-select>
          <el-input v-model="query.provider" placeholder="Provider" clearable />
          <el-button type="primary" @click="search">查询</el-button>
        </div>

        <el-table :data="logs" stripe>
          <el-table-column prop="provider" label="Provider" width="120" />
          <el-table-column prop="model" label="模型" min-width="180" show-overflow-tooltip />
          <el-table-column prop="operation" label="类型" width="100" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'success' ? 'success' : 'danger'" effect="light">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="耗时" width="100">
            <template #default="{ row }">{{ row.latency_ms }}ms</template>
          </el-table-column>
          <el-table-column label="Token" width="110">
            <template #default="{ row }">{{ row.prompt_tokens + row.completion_tokens }}</template>
          </el-table-column>
          <el-table-column label="时间" width="170">
            <template #default="{ row }">{{ timeText(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="detail = row">详情</el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-pagination
          v-model:current-page="query.page"
          v-model:page-size="query.page_size"
          layout="total, prev, pager, next"
          :total="total"
          @current-change="loadData"
        />
      </div>
    </section>

    <el-drawer :model-value="!!detail" title="调用详情" size="520px" @close="detail = undefined">
      <div v-if="detail" class="detail">
        <span>Provider</span><strong>{{ detail.provider }}</strong>
        <span>模型</span><strong>{{ detail.model || "-" }}</strong>
        <span>类型</span><strong>{{ detail.operation }}</strong>
        <span>耗时</span><strong>{{ detail.latency_ms }}ms</strong>
        <span>请求字符</span><strong>{{ detail.request_chars }}</strong>
        <span>响应字符</span><strong>{{ detail.response_chars }}</strong>
        <span>错误</span><p>{{ detail.error_message || "无" }}</p>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped lang="scss">
.logs-page {
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

.page-head h2,
.panel-title h3 {
  margin: 0;
}

.page-head p:last-child {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
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
.panel-title span,
.model-list span {
  color: var(--app-text-secondary);
}

.metric-card strong {
  font-size: 26px;
}

.layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 16px;
}

.panel {
  padding: 18px;
}

.panel-title,
.toolbar,
.model-list > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.toolbar {
  justify-content: flex-start;
  gap: 8px;
  margin-bottom: 14px;
}

.toolbar .el-select,
.toolbar .el-input {
  width: 150px;
}

.model-list {
  display: grid;
  gap: 12px;
  margin-top: 14px;
}

.model-list > div {
  gap: 12px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.model-list div div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.detail {
  display: grid;
  grid-template-columns: 96px 1fr;
  gap: 12px;
}

.detail span {
  color: var(--app-text-secondary);
}

.detail p {
  grid-column: span 2;
  padding: 12px;
  margin: 0;
  white-space: pre-wrap;
  background: var(--app-fill);
  border-radius: 8px;
}

@media (max-width: 1100px) {
  .metric-grid,
  .layout {
    grid-template-columns: 1fr 1fr;
  }

  .panel.main {
    grid-column: span 2;
  }
}
</style>
