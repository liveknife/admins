<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import { getSystemHealth, type SystemHealth } from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "SystemToolsHealth" });

const loading = ref(false);
const health = ref<SystemHealth>();

const statusText = computed(() => (health.value?.status === "healthy" ? "运行正常" : "需要关注"));
const statusClass = computed(() => (health.value?.status === "healthy" ? "is-healthy" : "is-warning"));

const cards = computed(() => [
  { label: "运行协程", value: health.value?.cpu.value ?? 0, unit: health.value?.cpu.unit ?? "", tone: "blue" },
  { label: "内存占用", value: (health.value?.memory.value ?? 0).toFixed(1), unit: health.value?.memory.unit ?? "MB", tone: "green" },
  { label: "接口均耗时", value: (health.value?.api.average_ms ?? 0).toFixed(1), unit: "ms", tone: "violet" },
  { label: "慢请求", value: health.value?.api.slow_requests ?? 0, unit: "次", tone: "orange" }
]);

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getSystemHealth();
    health.value = res.health;
  } catch {
    message("系统健康数据加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="health-page" v-loading="loading">
    <section class="health-hero" :class="statusClass">
      <div>
        <p class="eyebrow">Operations radar</p>
        <h2>系统健康监控</h2>
        <p>监控运行时、接口耗时、数据库连接和 WebSocket 在线状态。</p>
      </div>
      <div class="radar-status">
        <span class="pulse-dot" />
        <strong>{{ statusText }}</strong>
        <small v-if="health">{{ dayjs(health.checked_at).format("YYYY-MM-DD HH:mm:ss") }}</small>
      </div>
    </section>

    <section class="metric-grid">
      <div v-for="item in cards" :key="item.label" class="metric-card" :class="`tone-${item.tone}`">
        <span>{{ item.label }}</span>
        <strong>{{ item.value }}</strong>
        <small>{{ item.unit }}</small>
      </div>
    </section>

    <section class="health-grid">
      <div class="panel">
        <div class="panel-title">数据库连接</div>
        <div class="kv-list">
          <span>状态</span><strong>{{ health?.database.status || "-" }}</strong>
          <span>Ping</span><strong>{{ health?.database.ping_ms ?? 0 }} ms</strong>
          <span>打开连接</span><strong>{{ health?.database.open_connection ?? 0 }}</strong>
          <span>使用中</span><strong>{{ health?.database.in_use ?? 0 }}</strong>
          <span>空闲</span><strong>{{ health?.database.idle ?? 0 }}</strong>
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">WebSocket 在线状态</div>
        <div class="socket-box">
          <div>
            <span>在线用户</span>
            <strong>{{ health?.websocket.online_users ?? 0 }}</strong>
          </div>
          <div>
            <span>连接数</span>
            <strong>{{ health?.websocket.connections ?? 0 }}</strong>
          </div>
        </div>
      </div>

      <div class="panel full">
        <div class="panel-title">接口耗时排行</div>
        <el-table :data="health?.api.top_paths ?? []" stripe>
          <el-table-column prop="method" label="方法" width="90" />
          <el-table-column prop="path" label="接口" min-width="260" show-overflow-tooltip />
          <el-table-column prop="count" label="请求数" width="110" />
          <el-table-column label="平均耗时" width="140">
            <template #default="{ row }">{{ row.average_ms.toFixed(1) }} ms</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel full">
        <div class="panel-title">异常提示</div>
        <div class="alert-list">
          <el-alert
            v-for="item in health?.alerts ?? []"
            :key="item"
            :title="item"
            type="warning"
            show-icon
            :closable="false"
          />
          <el-empty v-if="!(health?.alerts?.length)" description="暂无异常提示" />
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.health-page {
  padding: 24px;
  display: grid;
  gap: 16px;
}

.health-hero,
.metric-card,
.panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.health-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 24px;
  border-left: 5px solid var(--app-green);
}

.health-hero.is-warning {
  border-left-color: var(--app-orange);
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-green);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.health-hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 760;
}

.health-hero p:last-child {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.radar-status {
  display: grid;
  justify-items: end;
  gap: 4px;
}

.pulse-dot {
  width: 12px;
  height: 12px;
  background: var(--app-green);
  border-radius: 50%;
  box-shadow: 0 0 0 8px rgb(22 163 74 / 12%);
}

.is-warning .pulse-dot {
  background: var(--app-orange);
  box-shadow: 0 0 0 8px rgb(217 119 6 / 14%);
}

.radar-status strong {
  font-size: 18px;
}

.radar-status small {
  color: var(--app-text-muted);
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card {
  padding: 18px;
  display: grid;
  gap: 6px;
}

.metric-card span,
.metric-card small {
  color: var(--app-text-secondary);
}

.metric-card strong {
  font-size: 28px;
  color: var(--app-text);
}

.tone-blue { border-top: 3px solid var(--app-primary); }
.tone-green { border-top: 3px solid var(--app-green); }
.tone-violet { border-top: 3px solid var(--app-violet); }
.tone-orange { border-top: 3px solid var(--app-orange); }

.health-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel {
  padding: 18px;
}

.panel.full {
  grid-column: 1 / -1;
}

.panel-title {
  margin-bottom: 14px;
  color: var(--app-text);
  font-weight: 760;
}

.kv-list {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 10px 16px;
  color: var(--app-text-secondary);
}

.kv-list strong {
  color: var(--app-text);
}

.socket-box {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 12px;
}

.socket-box div {
  padding: 16px;
  background: var(--app-surface-muted);
  border-radius: 8px;
}

.socket-box span {
  display: block;
  color: var(--app-text-secondary);
}

.socket-box strong {
  display: block;
  margin-top: 8px;
  font-size: 28px;
}

.alert-list {
  display: grid;
  gap: 10px;
}

@media (max-width: 980px) {
  .metric-grid,
  .health-grid {
    grid-template-columns: 1fr;
  }

  .health-hero {
    align-items: flex-start;
    flex-direction: column;
    gap: 14px;
  }

  .radar-status {
    justify-items: start;
  }
}
</style>
