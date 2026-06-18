<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import echarts from "@/plugins/echarts";
import { getAdminDashboard, type DashboardSummary } from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "Welcome" });

type ChartKey = "trend" | "composition" | "operations" | "notifications" | "score";

const router = useRouter();
const loading = ref(false);
const dashboard = ref<DashboardSummary>();

const trendChartEl = ref<HTMLDivElement | null>(null);
const compositionChartEl = ref<HTMLDivElement | null>(null);
const operationsChartEl = ref<HTMLDivElement | null>(null);
const notificationsChartEl = ref<HTMLDivElement | null>(null);
const scoreChartEl = ref<HTMLDivElement | null>(null);
const chartRefs = {
  trend: trendChartEl,
  composition: compositionChartEl,
  operations: operationsChartEl,
  notifications: notificationsChartEl,
  score: scoreChartEl
};
const charts = new Map<ChartKey, echarts.ECharts>();
let themeObserver: MutationObserver | undefined;

const formatNumber = (value = 0) => new Intl.NumberFormat("zh-CN").format(value);

const summaryCards = computed(() => [
  {
    label: "用户总数",
    value: dashboard.value?.user_count ?? 0,
    icon: "ri:user-3-line",
    tone: "cyan",
    hint: "已录入账号"
  },
  {
    label: "活跃用户",
    value: dashboard.value?.active_user_count ?? 0,
    icon: "ri:user-smile-line",
    tone: "green",
    hint: "可用账号"
  },
  {
    label: "消息总量",
    value: dashboard.value?.message_count ?? 0,
    icon: "ri:chat-3-line",
    tone: "violet",
    hint: "聊天消息"
  },
  {
    label: "未读通知",
    value: dashboard.value?.unread_notification ?? 0,
    icon: "ri:notification-3-line",
    tone: "orange",
    hint: "等待处理"
  }
]);

const systemScore = computed(() => {
  const data = dashboard.value;
  if (!data) return 0;
  const activeRatio = data.user_count ? data.active_user_count / data.user_count : 1;
  const unreadPenalty = Math.min(data.unread_notification * 4, 26);
  const permissionScore = data.permission_count > 0 ? 18 : 8;
  return Math.max(62, Math.min(99, Math.round(activeRatio * 55 + permissionScore + 24 - unreadPenalty)));
});

const operationStats = computed(() => {
  const stats = new Map<string, number>();
  for (const item of dashboard.value?.recent_logs ?? []) {
    stats.set(item.action || "系统操作", (stats.get(item.action || "系统操作") ?? 0) + 1);
  }
  return [...stats.entries()].map(([name, value]) => ({ name, value }));
});

const notificationStats = computed(() => {
  const rows = dashboard.value?.recent_notifications ?? [];
  const unread = rows.filter(item => !item.is_read).length;
  return [
    { name: "未读", value: unread },
    { name: "已读", value: Math.max(rows.length - unread, 0) }
  ];
});

const loadDashboard = async () => {
  loading.value = true;
  try {
    const res = await getAdminDashboard();
    dashboard.value = res.dashboard;
  } catch {
    message("仪表盘数据加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const goTo = (path: string) => router.push(path);

const isDarkTheme = () => document.documentElement.classList.contains("dark");

const chartTheme = () => {
  const dark = isDarkTheme();
  return {
    text: dark ? "#a6b6ca" : "#59677a",
    title: dark ? "#edf5ff" : "#162033",
    grid: dark ? "rgba(166, 182, 202, 0.16)" : "rgba(89, 103, 122, 0.14)",
    axis: dark ? "#33475f" : "#dbe5f0",
    tooltipBg: dark ? "rgba(11, 22, 36, 0.94)" : "rgba(255, 255, 255, 0.96)",
    tooltipBorder: dark ? "#2c4058" : "#dbe5f0",
    tooltipText: dark ? "#edf5ff" : "#162033",
    panel: dark ? "#101b2a" : "#ffffff",
    cyan: dark ? "#2dd4f7" : "#09b6d7",
    green: dark ? "#34d399" : "#16a679",
    violet: dark ? "#a78bfa" : "#7c5cff",
    orange: dark ? "#fbbf24" : "#f59e0b",
    red: dark ? "#fb7185" : "#ef4444"
  };
};

const baseText = () => ({
  color: chartTheme().text,
  fontFamily: "Inter, Arial, sans-serif"
});

const baseTooltip = () => {
  const theme = chartTheme();
  return {
    backgroundColor: theme.tooltipBg,
    borderColor: theme.tooltipBorder,
    textStyle: { color: theme.tooltipText }
  };
};

const buildTrendOption = () => {
  const rows = dashboard.value?.metric_trend ?? [];
  const labels = rows.map(item => dayjs(item.date).format("MM-DD"));
  const theme = chartTheme();
  return {
    color: [theme.cyan, theme.violet, theme.green],
    tooltip: { trigger: "axis", ...baseTooltip() },
    legend: { top: 0, right: 0, textStyle: baseText(), itemWidth: 10, itemHeight: 10 },
    grid: { left: 8, right: 10, top: 42, bottom: 8, containLabel: true },
    xAxis: {
      type: "category",
      boundaryGap: false,
      data: labels,
      axisLine: { lineStyle: { color: theme.axis } },
      axisTick: { show: false },
      axisLabel: { color: theme.text }
    },
    yAxis: {
      type: "value",
      minInterval: 1,
      splitLine: { lineStyle: { color: theme.grid } },
      axisLabel: { color: theme.text }
    },
    series: [
      {
        name: "新增用户",
        type: "line",
        smooth: true,
        symbolSize: 7,
        areaStyle: { color: isDarkTheme() ? "rgba(45, 212, 247, 0.16)" : "rgba(9, 182, 215, 0.12)" },
        data: rows.map(item => item.users)
      },
      {
        name: "聊天消息",
        type: "line",
        smooth: true,
        symbolSize: 7,
        areaStyle: { color: isDarkTheme() ? "rgba(167, 139, 250, 0.14)" : "rgba(124, 92, 255, 0.1)" },
        data: rows.map(item => item.messages)
      },
      {
        name: "操作日志",
        type: "bar",
        barWidth: 12,
        itemStyle: { borderRadius: [5, 5, 0, 0] },
        data: rows.map(item => item.logs)
      }
    ]
  };
};

const buildCompositionOption = () => {
  const data = dashboard.value;
  const theme = chartTheme();
  return {
    color: [theme.cyan, theme.green, theme.orange, theme.red],
    tooltip: { trigger: "item", ...baseTooltip() },
    legend: { bottom: 0, left: "center", textStyle: baseText(), itemWidth: 10, itemHeight: 10 },
    series: [
      {
        name: "资源构成",
        type: "pie",
        radius: ["52%", "72%"],
        center: ["50%", "45%"],
        avoidLabelOverlap: true,
        itemStyle: {
          borderColor: theme.panel,
          borderWidth: 4,
          borderRadius: 6
        },
        label: { color: theme.title, formatter: "{b}\n{c}" },
        data: [
          { name: "用户", value: data?.user_count ?? 0 },
          { name: "角色", value: data?.role_count ?? 0 },
          { name: "权限", value: data?.permission_count ?? 0 },
          { name: "消息", value: data?.message_count ?? 0 }
        ]
      }
    ]
  };
};

const buildOperationsOption = () => {
  const theme = chartTheme();
  return {
  color: [theme.cyan],
  tooltip: { trigger: "axis", ...baseTooltip() },
  grid: { left: 8, right: 16, top: 12, bottom: 8, containLabel: true },
  xAxis: {
    type: "value",
    minInterval: 1,
    splitLine: { lineStyle: { color: theme.grid } },
    axisLabel: { color: theme.text }
  },
  yAxis: {
    type: "category",
    data: operationStats.value.map(item => item.name),
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: { color: theme.title, width: 86, overflow: "truncate" }
  },
  series: [
    {
      name: "次数",
      type: "bar",
      barWidth: 12,
      data: operationStats.value.map(item => item.value),
      itemStyle: {
        borderRadius: [0, 8, 8, 0],
        color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
          { offset: 0, color: theme.violet },
          { offset: 1, color: theme.cyan }
        ])
      }
    }
  ]
};
};

const buildNotificationsOption = () => {
  const theme = chartTheme();
  return {
  color: [theme.red, theme.green],
  tooltip: { trigger: "item", ...baseTooltip() },
  series: [
    {
      name: "通知状态",
      type: "pie",
      radius: ["58%", "78%"],
      center: ["50%", "50%"],
      label: { color: theme.title, formatter: "{b} {c}" },
      itemStyle: {
        borderColor: theme.panel,
        borderWidth: 4,
        borderRadius: 6
      },
      data: notificationStats.value
    }
  ],
  graphic: {
    type: "text",
    left: "center",
    top: "middle",
    style: {
      text: `${dashboard.value?.unread_notification ?? 0}\n未读`,
      fill: theme.title,
      fontSize: 18,
      fontWeight: 700,
      lineHeight: 24,
      align: "center"
    }
  }
};
};

const buildScoreOption = () => {
  const theme = chartTheme();
  return {
  series: [
    {
      type: "gauge",
      startAngle: 210,
      endAngle: -30,
      center: ["50%", "58%"],
      radius: "92%",
      min: 0,
      max: 100,
      progress: {
        show: true,
        width: 14,
        roundCap: true,
        itemStyle: { color: theme.cyan }
      },
      axisLine: {
        lineStyle: {
          width: 14,
          color: [
            [0.72, isDarkTheme() ? "rgba(251, 113, 133, 0.24)" : "rgba(239, 68, 68, 0.16)"],
            [0.9, isDarkTheme() ? "rgba(251, 191, 36, 0.24)" : "rgba(245, 158, 11, 0.16)"],
            [1, isDarkTheme() ? "rgba(45, 212, 247, 0.24)" : "rgba(9, 182, 215, 0.16)"]
          ]
        }
      },
      axisTick: { show: false },
      splitLine: { show: false },
      axisLabel: { show: false },
      pointer: { show: false },
      anchor: { show: false },
      detail: {
        valueAnimation: true,
        offsetCenter: [0, "18%"],
        fontSize: 18,
        fontWeight: 800,
        color: theme.title,
        formatter: "{value}"
      },
      title: {
        offsetCenter: [0, "-8%"],
        color: theme.text,
        fontSize: 13
      },
      data: [{ value: systemScore.value, name: "运行评分" }]
    }
  ]
};
};

const renderCharts = () => {
  const options = {
    trend: buildTrendOption(),
    composition: buildCompositionOption(),
    operations: buildOperationsOption(),
    notifications: buildNotificationsOption(),
    score: buildScoreOption()
  };

  (Object.keys(chartRefs) as ChartKey[]).forEach(key => {
    const el = chartRefs[key].value;
    if (!el) return;
    const chart = charts.get(key) ?? echarts.init(el);
    charts.set(key, chart);
    chart.setOption(options[key], true);
  });
};

const resizeCharts = () => {
  charts.forEach(chart => chart.resize());
};

watch(
  dashboard,
  async () => {
    await nextTick();
    renderCharts();
  },
  { deep: true }
);

onMounted(async () => {
  await loadDashboard();
  window.addEventListener("resize", resizeCharts);
  themeObserver = new MutationObserver(() => renderCharts());
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class", "data-theme", "style"]
  });
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", resizeCharts);
  themeObserver?.disconnect();
  charts.forEach(chart => chart.dispose());
  charts.clear();
});
</script>

<template>
  <div class="dashboard-page" v-loading="loading">
    <section class="dashboard-hero">
      <div class="hero-copy">
        <div class="hero-kicker">
          <span class="signal-dot" />
          管理中枢在线
        </div>
        <h1>仪表盘首页</h1>
        <p>聚合用户、消息、通知与操作日志，快速判断后台系统今天是否平稳。</p>
        <div class="hero-actions">
          <el-button type="primary" @click="loadDashboard">
            <i class="ri-refresh-line" />
            刷新数据
          </el-button>
          <el-button @click="goTo('/system-tools/health')">
            <i class="ri-pulse-line" />
            查看健康监控
          </el-button>
        </div>
      </div>
      <div class="hero-radar" aria-label="系统态势">
        <div ref="scoreChartEl" class="score-chart" />
      </div>
    </section>

    <section class="metric-grid">
      <article v-for="item in summaryCards" :key="item.label" class="metric-card" :class="`tone-${item.tone}`">
        <div class="metric-icon">
          <i :class="item.icon" />
        </div>
        <div class="metric-body">
          <span>{{ item.label }}</span>
          <strong>{{ formatNumber(item.value) }}</strong>
          <small>{{ item.hint }}</small>
        </div>
      </article>
    </section>

    <section class="chart-grid">
      <article class="dashboard-panel panel-wide">
        <div class="panel-head">
          <div>
            <span>近 7 天趋势</span>
            <h2>增长、消息与操作波形</h2>
          </div>
          <el-tag effect="dark" type="success">ECharts</el-tag>
        </div>
        <div ref="trendChartEl" class="chart-box" />
      </article>

      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <span>资源构成</span>
            <h2>核心资产占比</h2>
          </div>
        </div>
        <div ref="compositionChartEl" class="chart-box" />
      </article>

      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <span>操作分布</span>
            <h2>最近日志动作</h2>
          </div>
          <el-button link type="primary" @click="goTo('/go-admin/operation-logs')">查看全部</el-button>
        </div>
        <div ref="operationsChartEl" class="chart-box compact" />
      </article>

      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <span>通知状态</span>
            <h2>待处理提醒</h2>
          </div>
          <el-button link type="primary" @click="goTo('/go-admin/notifications')">通知中心</el-button>
        </div>
        <div ref="notificationsChartEl" class="chart-box compact" />
      </article>
    </section>

    <section class="activity-grid">
      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <span>操作日志</span>
            <h2>最近管理员动作</h2>
          </div>
        </div>
        <div class="activity-list">
          <div v-for="item in dashboard?.recent_logs ?? []" :key="item.id" class="activity-item">
            <div class="activity-mark log" />
            <div class="activity-main">
              <strong>{{ item.action || "系统操作" }}</strong>
              <p>{{ item.username || "系统" }} · {{ item.resource || "后台资源" }} · {{ item.detail || "暂无详情" }}</p>
            </div>
            <time>{{ dayjs(item.created_at).format("MM-DD HH:mm") }}</time>
          </div>
          <el-empty v-if="!dashboard?.recent_logs?.length" description="暂无操作日志" :image-size="80" />
        </div>
      </article>

      <article class="dashboard-panel">
        <div class="panel-head">
          <div>
            <span>通知中心</span>
            <h2>最新系统消息</h2>
          </div>
        </div>
        <div class="activity-list">
          <div v-for="item in dashboard?.recent_notifications ?? []" :key="item.id" class="activity-item">
            <div class="activity-mark notice" :class="{ unread: !item.is_read }" />
            <div class="activity-main">
              <strong>
                <el-tag :type="item.is_read ? 'info' : 'danger'" size="small">
                  {{ item.is_read ? "已读" : "未读" }}
                </el-tag>
                {{ item.title }}
              </strong>
              <p>{{ item.content }}</p>
            </div>
            <time>{{ dayjs(item.created_at).format("MM-DD HH:mm") }}</time>
          </div>
          <el-empty v-if="!dashboard?.recent_notifications?.length" description="暂无通知" :image-size="80" />
        </div>
      </article>
    </section>
  </div>
</template>

<style scoped lang="scss">
.dashboard-page {
  min-height: 100%;
  padding: 22px;
  display: grid;
  gap: 18px;
  color: var(--app-text);
  background:
    radial-gradient(circle at 8% 0%, color-mix(in srgb, var(--app-cyan) 15%, transparent), transparent 28%),
    radial-gradient(circle at 92% 2%, color-mix(in srgb, var(--app-violet) 13%, transparent), transparent 30%),
    var(--app-bg);
}

.dashboard-hero,
.dashboard-panel,
.metric-card {
  border: 1px solid rgba(118, 168, 215, 0.2);
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-surface) 92%, transparent);
  box-shadow: var(--app-shadow-soft);
  backdrop-filter: blur(14px);
}

.dashboard-hero {
  position: relative;
  overflow: hidden;
  min-height: 248px;
  padding: 30px;
  display: flex;
  align-items: center;
  justify-content: space-between;

  &::before {
    content: "";
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(rgba(49, 215, 255, 0.09) 1px, transparent 1px),
      linear-gradient(90deg, rgba(49, 215, 255, 0.09) 1px, transparent 1px);
    background-size: 46px 46px;
    mask-image: linear-gradient(90deg, #000 0%, transparent 70%);
    pointer-events: none;
  }
}

.hero-copy {
  position: relative;
  z-index: 1;
  max-width: 620px;

  h1 {
    margin: 10px 0 8px;
    font-size: clamp(32px, 5vw, 54px);
    line-height: 1;
    font-weight: 850;
    letter-spacing: 0;
    color: var(--app-text);
  }

  p {
    margin: 0;
    max-width: 560px;
    color: var(--app-text-secondary);
    font-size: 15px;
    line-height: 1.8;
  }
}

.hero-kicker {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--app-cyan);
  font-size: 13px;
  font-weight: 700;
}

.signal-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--app-green);
  box-shadow: 0 0 18px color-mix(in srgb, var(--app-green) 55%, transparent);
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 22px;

  :deep(.el-button) {
    border-radius: 8px;
  }

  i {
    margin-right: 6px;
  }
}

.hero-radar {
  position: relative;
  z-index: 1;
  width: 230px;
  height: 230px;
  flex: 0 0 auto;
}

.score-chart {
  width: 100%;
  height: 100%;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 14px;
}

.metric-card {
  position: relative;
  overflow: hidden;
  padding: 18px;
  display: flex;
  gap: 14px;
  align-items: center;

  &::after {
    content: "";
    position: absolute;
    inset: auto 16px 0 16px;
    height: 3px;
    border-radius: 3px 3px 0 0;
    background: var(--card-accent);
  }
}

.tone-cyan { --card-accent: var(--app-cyan); }
.tone-green { --card-accent: var(--app-green); }
.tone-violet { --card-accent: var(--app-violet); }
.tone-orange { --card-accent: var(--app-orange); }

.metric-icon {
  width: 48px;
  height: 48px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--card-accent);
  font-size: 24px;
  background: color-mix(in srgb, var(--card-accent) 14%, transparent);
}

.metric-body {
  display: grid;
  gap: 2px;

  span,
  small {
    color: var(--app-text-muted);
    font-size: 13px;
  }

  strong {
    color: var(--app-text);
    font-size: 30px;
    line-height: 1.1;
  }
}

.chart-grid,
.activity-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 18px;
}

.panel-wide {
  grid-column: span 2;
}

.dashboard-panel {
  padding: 18px;
  min-width: 0;
}

.panel-head {
  min-height: 42px;
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;

  span {
    display: block;
    color: var(--app-cyan);
    font-size: 12px;
    font-weight: 700;
  }

  h2 {
    margin: 4px 0 0;
    color: var(--app-text);
    font-size: 17px;
    line-height: 1.35;
    font-weight: 800;
    letter-spacing: 0;
  }
}

.chart-box {
  width: 100%;
  height: 340px;

  &.compact {
    height: 280px;
  }
}

.activity-list {
  display: grid;
  gap: 10px;
}

.activity-item {
  display: grid;
  grid-template-columns: 10px 1fr auto;
  gap: 12px;
  align-items: flex-start;
  padding: 12px;
  border: 1px solid var(--app-border-soft);
  border-radius: 8px;
  background: var(--app-surface-soft);
}

.activity-mark {
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--app-cyan);
  box-shadow: 0 0 14px color-mix(in srgb, var(--app-cyan) 50%, transparent);

  &.notice {
    background: var(--app-green);
    box-shadow: 0 0 14px color-mix(in srgb, var(--app-green) 45%, transparent);
  }

  &.unread {
    background: var(--app-red);
    box-shadow: 0 0 14px color-mix(in srgb, var(--app-red) 48%, transparent);
  }
}

.activity-main {
  min-width: 0;

  strong {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--app-text);
    font-size: 14px;
  }

  p {
    margin: 5px 0 0;
    color: var(--app-text-secondary);
    font-size: 13px;
    line-height: 1.55;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

time {
  color: var(--app-text-muted);
  font-size: 12px;
  white-space: nowrap;
}

:deep(.el-loading-mask) {
  background-color: color-mix(in srgb, var(--app-bg) 72%, transparent);
}

:deep(.el-empty__description p) {
  color: var(--app-text-muted);
}

@media (max-width: 1200px) {
  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 900px) {
  .dashboard-page {
    padding: 14px;
  }

  .dashboard-hero {
    flex-direction: column;
    align-items: flex-start;
  }

  .hero-radar {
    width: 190px;
    height: 190px;
    align-self: center;
  }

  .chart-grid,
  .activity-grid {
    grid-template-columns: 1fr;
  }

  .panel-wide {
    grid-column: span 1;
  }
}

@media (max-width: 560px) {
  .metric-grid {
    grid-template-columns: 1fr;
  }

  .activity-item {
    grid-template-columns: 10px 1fr;

    time {
      grid-column: 2;
    }
  }
}
</style>
