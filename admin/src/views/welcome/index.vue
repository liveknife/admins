<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import echarts from "@/plugins/echarts";
import { getAdminDashboard, type DashboardSummary } from "@/api/admin";

dayjs.extend(relativeTime);

defineOptions({ name: "Welcome" });

type ChartKey =
  | "trend"
  | "source"
  | "resource"
  | "message"
  | "score"
  | "sparkUsers"
  | "sparkActive"
  | "sparkMessages"
  | "sparkUnread";

type Tone = "cyan" | "green" | "violet" | "amber";
type MessageStat = { name: string; value: number };

const router = useRouter();
const loading = ref(false);
const dashboard = ref<DashboardSummary>();

const trendEl = ref<HTMLDivElement | null>(null);
const sourceEl = ref<HTMLDivElement | null>(null);
const resourceEl = ref<HTMLDivElement | null>(null);
const messageEl = ref<HTMLDivElement | null>(null);
const scoreEl = ref<HTMLDivElement | null>(null);
const sparkUsersEl = ref<HTMLDivElement | null>(null);
const sparkActiveEl = ref<HTMLDivElement | null>(null);
const sparkMessagesEl = ref<HTMLDivElement | null>(null);
const sparkUnreadEl = ref<HTMLDivElement | null>(null);

const charts = new Map<ChartKey, echarts.ECharts>();
let themeObserver: MutationObserver | undefined;
let resizeObserver: ResizeObserver | undefined;

const fmt = (value = 0) => new Intl.NumberFormat("zh-CN").format(value);
const isDark = () => document.documentElement.classList.contains("dark");
const dashboardData = computed(() => dashboard.value ?? fallbackDashboard);

const palette = () => {
  const dark = isDark();
  return {
    page: dark ? "#061022" : "#eef5ff",
    panel: dark ? "rgba(9, 22, 44, .82)" : "rgba(255, 255, 255, .9)",
    border: dark ? "rgba(55, 133, 255, .28)" : "rgba(49, 116, 218, .18)",
    title: dark ? "#f4f8ff" : "#12213a",
    text: dark ? "#aebfdb" : "#586a86",
    muted: dark ? "#71809b" : "#7a8aa5",
    grid: dark ? "rgba(119, 154, 208, .16)" : "rgba(76, 101, 140, .14)",
    blue: dark ? "#1686ff" : "#1677ff",
    cyan: dark ? "#22d3ee" : "#09b6d8",
    green: dark ? "#2ee59d" : "#10b981",
    violet: dark ? "#8b5cf6" : "#7c3aed",
    amber: dark ? "#fbbf24" : "#f59e0b",
    track: dark ? "rgba(64, 90, 132, .38)" : "rgba(170, 187, 212, .45)",
    tooltip: dark ? "rgba(5, 15, 31, .96)" : "rgba(255, 255, 255, .98)"
  };
};

const textStyle = () => ({ color: palette().text, fontFamily: "Inter, Arial, sans-serif" });
const tooltipStyle = () => ({
  backgroundColor: palette().tooltip,
  borderColor: palette().border,
  textStyle: { color: palette().title }
});

const userName = computed(() => "admin");
const todayText = computed(() => dayjs().format("YYYY年MM月DD日 dddd"));

const systemScore = computed(() => {
  const data = dashboardData.value;
  const activeRate = data.user_count ? data.active_user_count / data.user_count : 1;
  const noticePenalty = Math.min(data.unread_notification * 4, 24);
  const permissionBonus = data.permission_count > 0 ? 16 : 8;
  return Math.max(60, Math.min(99, Math.round(activeRate * 58 + permissionBonus + 22 - noticePenalty)));
});

const scoreLabel = computed(() => {
  if (systemScore.value >= 85) return "优秀";
  if (systemScore.value >= 70) return "良好";
  return "需关注";
});

const heroBadges = computed(() => [
  { label: "消息总量", value: dashboardData.value.message_count, icon: "ri:chat-smile-3-line", tone: "violet" as Tone },
  { label: "活跃用户", value: dashboardData.value.active_user_count, icon: "ri:user-heart-line", tone: "green" as Tone },
  { label: "未读通知", value: dashboardData.value.unread_notification, icon: "ri:notification-3-line", tone: "amber" as Tone },
  { label: "用户总数", value: dashboardData.value.user_count, icon: "ri:user-3-line", tone: "cyan" as Tone }
]);

const metricCards = computed(() => [
  {
    label: "用户总数",
    value: dashboardData.value.user_count,
    hint: "已录入账号",
    icon: "ri:user-3-line",
    tone: "cyan" as Tone,
    trend: "+12%",
    chart: "sparkUsers" as ChartKey
  },
  {
    label: "活跃用户",
    value: dashboardData.value.active_user_count,
    hint: "可用账号",
    icon: "ri:user-heart-line",
    tone: "green" as Tone,
    trend: "+8%",
    chart: "sparkActive" as ChartKey
  },
  {
    label: "消息总量",
    value: dashboardData.value.message_count,
    hint: "聊天消息",
    icon: "ri:message-3-line",
    tone: "violet" as Tone,
    trend: "+18%",
    chart: "sparkMessages" as ChartKey
  },
  {
    label: "未读通知",
    value: dashboardData.value.unread_notification,
    hint: "待处理",
    icon: "ri:alarm-warning-line",
    tone: "amber" as Tone,
    trend: "+0%",
    chart: "sparkUnread" as ChartKey
  }
]);

const sourceStats = computed(() => {
  const rows = dashboardData.value.source_stats;
  if (rows?.length) return rows;
  return [
    { name: "直接访问", value: 42 },
    { name: "搜索引擎", value: 35 },
    { name: "外部链接", value: 28 },
    { name: "社交媒体", value: 15 },
    { name: "其他", value: 8 }
  ];
});

const sourceTotal = computed(() => sourceStats.value.reduce((sum, item) => sum + item.value, 0));

const sourceLegendRows = computed(() =>
  sourceStats.value.map((item, index) => ({
    ...item,
    tone: ["blue", "violet", "green", "amber", "cyan"][index % 5],
    percent: sourceTotal.value ? (item.value / sourceTotal.value) * 100 : 0
  }))
);

const fallbackMessageStats: MessageStat[] = [
  { name: "系统消息", value: 20 },
  { name: "聊天消息", value: 15 },
  { name: "通知公告", value: 6 },
  { name: "其他消息", value: 2 }
];

const normalizeMessageStats = (rows?: MessageStat[]) => {
  const buckets = new Map(fallbackMessageStats.map(item => [item.name, 0]));
  const systemTypes = new Set(["info", "success", "warning", "danger", "error", "system"]);
  const announcementTypes = new Set(["notice", "announcement", "公告", "通知"]);

  for (const row of rows ?? []) {
    const name = String(row.name ?? "").trim();
    const value = Number(row.value) || 0;
    if (!name || value <= 0) continue;

    if (buckets.has(name)) {
      buckets.set(name, (buckets.get(name) ?? 0) + value);
    } else if (systemTypes.has(name.toLowerCase())) {
      buckets.set("系统消息", (buckets.get("系统消息") ?? 0) + value);
    } else if (announcementTypes.has(name.toLowerCase())) {
      buckets.set("通知公告", (buckets.get("通知公告") ?? 0) + value);
    } else {
      buckets.set("其他消息", (buckets.get("其他消息") ?? 0) + value);
    }
  }

  const normalized = fallbackMessageStats.map(item => ({
    name: item.name,
    value: buckets.get(item.name) ?? 0
  }));

  return normalized.some(item => item.value > 0) ? normalized : fallbackMessageStats;
};

const messageStats = computed(() => normalizeMessageStats(dashboardData.value.message_type_stats));

const messageTotal = computed(() => messageStats.value.reduce((sum, item) => sum + item.value, 0));

const messageLegendRows = computed(() =>
  messageStats.value.map((item, index) => ({
    ...item,
    percent: messageTotal.value ? (item.value / messageTotal.value) * 100 : 0,
    tone: ["blue", "cyan", "amber", "violet"][index % 4]
  }))
);

const resourceStats = computed(() => {
  const resources = dashboardData.value.system_resources;
  return {
    cpu: Math.round(resources?.cpu_usage ?? 32),
    memory: Math.round(resources?.memory_usage ?? 48),
    disk: Math.round(resources?.disk_usage ?? 65),
    cpuTrend: resources?.cpu_trend?.length ? resources.cpu_trend : wave(24, 34, 18),
    memoryTrend: resources?.memory_trend?.length ? resources.memory_trend : wave(24, 46, 16)
  };
});

const recentActivities = computed(() => {
  const notices = dashboardData.value.recent_notifications ?? [];
  return notices.slice(0, 4).map((item, index) => ({
    id: item.id,
    title: item.title,
    content: item.content,
    time: dayjs(item.created_at).fromNow(),
    icon: ["ri:user-add-line", "ri:chat-smile-2-line", "ri:shield-check-line", "ri:megaphone-line"][index % 4],
    tone: ["blue", "cyan", "amber", "violet"][index % 4]
  }));
});

const fallbackDashboard: DashboardSummary = {
  user_count: 5,
  active_user_count: 3,
  role_count: 2,
  permission_count: 18,
  message_count: 43,
  unread_notification: 0,
  recent_logs: [
    logRow(1, "admin", "登录系统", "登录系统", 10),
    logRow(2, "admin", "更新用户信息", "用户管理", 18),
    logRow(3, "test", "发送消息", "聊天", 25),
    logRow(4, "admin", "发布公告", "后台公告", 35)
  ],
  recent_notifications: [
    noticeRow(1, "新用户注册", "demo 用户已注册成功", "success", 10),
    noticeRow(2, "收到新的聊天消息", "来自 test 的新消息", "info", 15),
    noticeRow(3, "系统更新完成", "系统版本更新到 v2.1.0", "success", 30),
    noticeRow(4, "新的公告发布", "系统维护通知", "warning", 60)
  ],
  metric_trend: [
    { date: "2024-12-26", users: 7, messages: 20, logs: 3 },
    { date: "2024-12-27", users: 15, messages: 31, logs: 10 },
    { date: "2024-12-28", users: 9, messages: 24, logs: 5 },
    { date: "2024-12-29", users: 8, messages: 20, logs: 4 },
    { date: "2024-12-30", users: 14, messages: 32, logs: 9 },
    { date: "2024-12-31", users: 15, messages: 29, logs: 7 },
    { date: "2025-01-01", users: 34, messages: 46, logs: 19 }
  ],
  system_resources: {
    cpu_usage: 32,
    memory_usage: 48,
    disk_usage: 65,
    cpu_trend: wave(24, 43, 14),
    memory_trend: wave(24, 30, 12)
  },
  source_stats: [
    { name: "直接访问", value: 42 },
    { name: "搜索引擎", value: 35 },
    { name: "外部链接", value: 28 },
    { name: "社交媒体", value: 15 },
    { name: "其他", value: 8 }
  ],
  message_type_stats: [
    { name: "系统消息", value: 20 },
    { name: "聊天消息", value: 15 },
    { name: "通知公告", value: 6 },
    { name: "其他消息", value: 2 }
  ]
};

function logRow(id: number, username: string, detail: string, resource: string, minutesAgo: number) {
  return {
    id,
    user_id: id,
    username,
    action: detail,
    resource,
    detail,
    ip: "127.0.0.1",
    user_agent: "Mozilla/5.0",
    created_at: dayjs().subtract(minutesAgo, "minute").toISOString()
  };
}

function noticeRow(id: number, title: string, content: string, type: string, minutesAgo: number) {
  return {
    id,
    title,
    content,
    type,
    is_read: minutesAgo > 20,
    created_at: dayjs().subtract(minutesAgo, "minute").toISOString()
  };
}

function wave(length: number, base: number, amplitude: number) {
  return Array.from({ length }, (_, index) =>
    Math.max(0, Math.round(base + Math.sin(index * 0.72) * amplitude + (index % 4) * 2))
  );
}

const chartEl = (key: ChartKey) => {
  const map: Record<ChartKey, typeof trendEl> = {
    trend: trendEl,
    source: sourceEl,
    resource: resourceEl,
    message: messageEl,
    score: scoreEl,
    sparkUsers: sparkUsersEl,
    sparkActive: sparkActiveEl,
    sparkMessages: sparkMessagesEl,
    sparkUnread: sparkUnreadEl
  };
  return map[key].value;
};

const setChart = (key: ChartKey, option: echarts.EChartsCoreOption) => {
  const el = chartEl(key);
  if (!el) return;
  const instance = charts.get(key) ?? echarts.init(el);
  charts.set(key, instance);
  instance.setOption(option, true);
};

const loadDashboard = async () => {
  loading.value = true;
  try {
    const res = await getAdminDashboard();
    if (res.dashboard) dashboard.value = res.dashboard;
  } catch (error) {
    console.warn("[Dashboard] load failed, fallback data is used.", error);
  } finally {
    loading.value = false;
    await nextTick();
    renderCharts();
  }
};

const goTo = (path: string) => router.push(path);

const renderCharts = () => {
  const t = palette();
  setChart("score", {
    series: [
      {
        type: "gauge",
        startAngle: 220,
        endAngle: -40,
        min: 0,
        max: 100,
        radius: "92%",
        center: ["50%", "54%"],
        progress: {
          show: true,
          width: 14,
          roundCap: true,
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 1, 1, [
              { offset: 0, color: t.blue },
              { offset: 1, color: t.cyan }
            ])
          }
        },
        axisLine: { lineStyle: { width: 14, color: [[1, t.track]] } },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        pointer: { show: false },
        detail: {
          valueAnimation: true,
          offsetCenter: [0, "8%"],
          formatter: "{value}",
          color: t.title,
          fontSize: 34,
          fontWeight: 800,
          lineHeight: 38
        },
        title: {
          offsetCenter: [0, "-22%"],
          color: t.text,
          fontSize: 12,
          lineHeight: 16
        },
        data: [{ value: systemScore.value, name: "系统运行评分" }]
      }
    ]
  });

  const trend = dashboardData.value.metric_trend ?? [];
  setChart("trend", {
    color: [t.blue, t.violet, t.green],
    tooltip: { trigger: "axis", ...tooltipStyle() },
    legend: { right: 6, top: 4, itemWidth: 14, itemHeight: 4, textStyle: textStyle() },
    grid: { left: 8, right: 12, top: 40, bottom: 8, containLabel: true },
    xAxis: {
      type: "category",
      boundaryGap: false,
      data: trend.map(item => dayjs(item.date).format("MM-DD")),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: t.grid } },
      axisLabel: { color: t.text, fontSize: 11 }
    },
    yAxis: {
      type: "value",
      minInterval: 1,
      splitLine: { lineStyle: { color: t.grid } },
      axisLabel: { color: t.text, fontSize: 11 }
    },
    series: [
      lineSeries("新增用户", trend.map(item => item.users), t.blue),
      lineSeries("聊天消息", trend.map(item => item.messages), t.violet),
      lineSeries("操作日志", trend.map(item => item.logs), t.green)
    ]
  });

  setChart("source", {
    color: [t.blue, t.violet, t.green, t.amber, t.cyan],
    tooltip: { trigger: "item", ...tooltipStyle(), formatter: "{b}<br />{c} ({d}%)" },
    graphic: {
      type: "text",
      left: "center",
      top: "center",
      style: {
        text: `总计\n${sourceTotal.value}`,
        fill: t.title,
        align: "center",
        fontSize: 17,
        fontWeight: 800,
        lineHeight: 24
      }
    },
    series: [
      {
        type: "pie",
        radius: ["48%", "76%"],
        center: ["50%", "50%"],
        label: { show: false },
        itemStyle: { borderColor: isDark() ? "#071226" : "#fff", borderWidth: 3, borderRadius: 5 },
        data: sourceStats.value
      }
    ]
  });

  setChart("resource", {
    tooltip: { ...tooltipStyle() },
    grid: { left: 8, right: 12, top: 166, bottom: 8, containLabel: true },
    xAxis: {
      type: "category",
      data: Array.from({ length: 24 }, (_, index) => `${String(index).padStart(2, "0")}:00`),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: t.grid } },
      axisLabel: { color: t.text, fontSize: 10 }
    },
    yAxis: {
      type: "value",
      max: 100,
      splitLine: { lineStyle: { color: t.grid } },
      axisLabel: { color: t.text, formatter: "{value}%" }
    },
    series: [
      gaugeSeries("CPU", resourceStats.value.cpu, ["18%", "36%"], t.green),
      gaugeSeries("内存", resourceStats.value.memory, ["50%", "36%"], t.cyan),
      gaugeSeries("磁盘", resourceStats.value.disk, ["82%", "36%"], t.blue),
      lineSeries("CPU", resourceStats.value.cpuTrend, t.blue),
      lineSeries("内存", resourceStats.value.memoryTrend, t.green)
    ]
  });

  const messageColors = [t.blue, t.cyan, t.amber, t.violet];
  setChart("message", {
    color: messageColors,
    tooltip: {
      trigger: "item",
      ...tooltipStyle(),
      formatter: (params: any) => {
        const value = Number(params.value) || 0;
        const percent = messageTotal.value ? ((value / messageTotal.value) * 100).toFixed(2) : "0.00";
        return `${params.name}<br />${value} (${percent}%)`;
      }
    },
    graphic: {
      type: "text",
      left: "center",
      top: "center",
      style: {
        text: `总计\n${messageTotal.value}`,
        fill: t.title,
        align: "center",
        fontSize: 18,
        fontWeight: 800,
        lineHeight: 26
      }
    },
    series: [
      {
        type: "pie",
        radius: ["52%", "76%"],
        center: ["50%", "50%"],
        avoidLabelOverlap: true,
        label: { show: false },
        itemStyle: {
          borderColor: isDark() ? "#071226" : "#fff",
          borderWidth: 3,
          borderRadius: 5,
          shadowBlur: 18,
          shadowColor: "rgba(22, 119, 255, .16)"
        },
        data: messageLegendRows.value.map((item, index) => ({
          name: item.name,
          value: item.value,
          itemStyle: { color: messageColors[index % messageColors.length] }
        }))
      }
    ]
  });

  metricCards.value.forEach(card => {
    const color = card.tone === "cyan" ? t.blue : card.tone === "green" ? t.green : card.tone === "violet" ? t.violet : t.amber;
    setChart(card.chart, sparkOption(wave(18, Math.max(card.value, 3), Math.max(card.value * 0.25, 2)), color));
  });

  charts.forEach(chart => chart.resize());
};

const lineSeries = (name: string, data: number[], color: string, xAxisIndex = 0) => ({
  name,
  type: "line",
  smooth: true,
  symbol: "circle",
  symbolSize: xAxisIndex ? 0 : 5,
  xAxisIndex,
  yAxisIndex: xAxisIndex,
  lineStyle: { width: 2, color },
  areaStyle: {
    color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
      { offset: 0, color: colorWithAlpha(color, 0.22) },
      { offset: 1, color: colorWithAlpha(color, 0) }
    ])
  },
  data
});

const gaugeSeries = (name: string, value: number, center: string[], color: string) => ({
  name,
  type: "gauge",
  startAngle: 220,
  endAngle: -40,
  center,
  radius: "32%",
  min: 0,
  max: 100,
  progress: { show: true, width: 8, roundCap: true, itemStyle: { color } },
  axisLine: { lineStyle: { width: 8, color: [[1, palette().track]] } },
  axisTick: { show: false },
  splitLine: { show: false },
  axisLabel: { show: false },
  pointer: { show: false },
  title: {
    offsetCenter: [0, "-56%"],
    color: palette().text,
    fontSize: 12,
    lineHeight: 16
  },
  detail: {
    formatter: "{value}%",
    offsetCenter: [0, "10%"],
    color: palette().title,
    fontSize: 22,
    fontWeight: 800,
    lineHeight: 26
  },
  data: [{ value, name }]
});

const sparkOption = (data: number[], color: string) => ({
  grid: { left: 0, right: 0, top: 3, bottom: 3 },
  xAxis: { type: "category", show: false, data: data.map((_, index) => index) },
  yAxis: { type: "value", show: false, min: Math.min(...data) * 0.82 },
  series: [
    {
      type: "line",
      smooth: true,
      symbol: "none",
      lineStyle: { width: 2, color },
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: colorWithAlpha(color, 0.22) },
          { offset: 1, color: colorWithAlpha(color, 0) }
        ])
      },
      data
    }
  ]
});

const colorWithAlpha = (hex: string, alpha: number) => {
  const value = hex.replace("#", "");
  const r = parseInt(value.slice(0, 2), 16);
  const g = parseInt(value.slice(2, 4), 16);
  const b = parseInt(value.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
};

onMounted(async () => {
  dashboard.value = fallbackDashboard;
  await nextTick();
  renderCharts();
  await loadDashboard();

  themeObserver = new MutationObserver(() => nextTick(renderCharts));
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ["class"]
  });

  resizeObserver = new ResizeObserver(() => charts.forEach(chart => chart.resize()));
  document.querySelectorAll(".dashboard-page .chart-host").forEach(el => resizeObserver?.observe(el));
});

onBeforeUnmount(() => {
  themeObserver?.disconnect();
  resizeObserver?.disconnect();
  charts.forEach(chart => chart.dispose());
  charts.clear();
});
</script>

<template>
  <div v-loading="loading" class="dashboard-page">
    <section class="hero-card">
      <div class="hero-copy">
        <div class="online-pill">
          <span />
          管理中枢在线
        </div>
        <h1>欢迎回来，{{ userName }}</h1>
        <p class="hero-date">今天是 {{ todayText }}</p>
        <p class="hero-desc">聚合用户、消息、通知与操作日志，快速判断后台系统今天是否平稳。</p>
        <div class="hero-actions">
          <el-button type="primary" @click="loadDashboard">
            <IconifyIconOnline icon="ri:refresh-line" />
            刷新数据
          </el-button>
          <el-button @click="goTo('/system-tools/health')">
            <IconifyIconOnline icon="ri:pulse-line" />
            查看健康监控
          </el-button>
        </div>
      </div>

      <div class="holo-stage" aria-hidden="true">
        <div class="holo-grid" />
        <div class="holo-base base-1" />
        <div class="holo-base base-2" />
        <div class="holo-cube">
          <span v-for="index in 6" :key="index" />
        </div>
        <i v-for="index in 8" :key="index" :class="`spark spark-${index}`" />
      </div>

      <div class="hero-badges">
        <article
          v-for="(badge, index) in heroBadges"
          :key="badge.label"
          class="float-badge"
          :class="`tone-${badge.tone}`"
          :style="{ '--delay': `${index * 70}ms` }"
        >
          <div class="float-icon">
            <IconifyIconOnline :icon="badge.icon" />
          </div>
          <span>{{ badge.label }}</span>
          <strong>{{ fmt(badge.value) }}</strong>
        </article>
      </div>

      <div class="score-card">
        <div ref="scoreEl" class="chart-host score-chart" />
        <span>{{ scoreLabel }}</span>
      </div>
    </section>

    <section class="metric-grid">
      <article
        v-for="card in metricCards"
        :key="card.label"
        class="metric-card"
        :class="`tone-${card.tone}`"
      >
        <div class="metric-icon">
          <IconifyIconOnline :icon="card.icon" />
        </div>
        <div class="metric-body">
          <span>{{ card.label }}</span>
          <strong>{{ fmt(card.value) }}</strong>
          <small>{{ card.hint }}</small>
        </div>
        <div
          :ref="el => {
            if (card.chart === 'sparkUsers') sparkUsersEl = el as HTMLDivElement;
            if (card.chart === 'sparkActive') sparkActiveEl = el as HTMLDivElement;
            if (card.chart === 'sparkMessages') sparkMessagesEl = el as HTMLDivElement;
            if (card.chart === 'sparkUnread') sparkUnreadEl = el as HTMLDivElement;
          }"
          class="chart-host sparkline"
        />
        <em>{{ card.trend }}</em>
      </article>
    </section>

    <section class="content-grid">
      <article class="panel trend-panel">
        <header>
          <div>
            <h2>趋势分析</h2>
            <p>近 7 天增长趋势</p>
          </div>
          <button type="button" @click="loadDashboard">查看更多</button>
        </header>
        <div ref="trendEl" class="chart-host chart-main" />
      </article>

      <article class="panel source-panel">
        <header>
          <div>
            <h2>访问来源分布</h2>
            <p>近 7 天统计</p>
          </div>
        </header>
        <div class="source-layout">
          <div ref="sourceEl" class="chart-host source-chart" />
          <div class="source-list">
            <div v-for="item in sourceLegendRows" :key="item.name" :class="`source-${item.tone}`">
              <i />
              <span>{{ item.name }}</span>
              <strong>{{ item.value }}</strong>
              <em>{{ item.percent.toFixed(2) }}%</em>
            </div>
          </div>
        </div>
      </article>

      <article class="panel resource-panel">
        <header>
          <div>
            <h2>系统资源监控</h2>
          </div>
        </header>
        <div ref="resourceEl" class="chart-host chart-main" />
      </article>

      <article class="panel logs-panel">
        <header>
          <div>
            <h2>最新操作日志</h2>
          </div>
          <button type="button" @click="goTo('/go-admin/operation-logs')">查看更多</button>
        </header>
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>时间</th>
                <th>用户</th>
                <th>操作内容</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in dashboardData.recent_logs.slice(0, 4)" :key="log.id">
                <td>{{ dayjs(log.created_at).format("YYYY-MM-DD HH:mm:ss") }}</td>
                <td>{{ log.username || "-" }}</td>
                <td>{{ log.detail || log.action }}</td>
                <td><span class="success-tag">成功</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </article>

      <article class="panel message-panel">
        <header>
          <div>
            <h2>消息类型统计</h2>
            <p>近 7 天统计</p>
          </div>
        </header>
        <div class="message-funnel">
          <div ref="messageEl" class="chart-host message-chart" />
          <div class="message-legend">
            <div v-for="item in messageLegendRows" :key="item.name" :class="`message-${item.tone}`">
              <i />
              <span>{{ item.name }}</span>
              <strong>{{ item.value }}</strong>
              <em>{{ item.percent.toFixed(2) }}%</em>
            </div>
          </div>
        </div>
      </article>

      <article class="panel activity-panel">
        <header>
          <div>
            <h2>实时动态</h2>
          </div>
        </header>
        <div class="activity-list">
          <div v-for="item in recentActivities" :key="item.id" class="activity-item">
            <div class="activity-icon" :class="`act-${item.tone}`">
              <IconifyIconOnline :icon="item.icon" />
            </div>
            <div>
              <strong>{{ item.title }}</strong>
              <p>{{ item.content }}</p>
            </div>
            <time>{{ item.time }}</time>
          </div>
        </div>
      </article>
    </section>
  </div>
</template>

<style scoped lang="scss">
.dashboard-page {
  --dash-page: #edf4ff;
  --dash-card: rgba(255, 255, 255, 0.88);
  --dash-card-strong: rgba(255, 255, 255, 0.96);
  --dash-border: rgba(41, 103, 190, 0.18);
  --dash-title: #14233b;
  --dash-text: #5a6d8d;
  --dash-muted: #8190a8;
  --dash-blue: #1677ff;
  --dash-cyan: #09b6d8;
  --dash-green: #10b981;
  --dash-violet: #7c3aed;
  --dash-amber: #f59e0b;
  --dash-shadow: 0 18px 46px rgba(35, 82, 138, 0.12);

  display: grid;
  gap: 14px;
  min-height: 100%;
  padding: 20px;
  color: var(--dash-title);
  background:
    radial-gradient(circle at 12% 0%, rgba(22, 119, 255, 0.1), transparent 30%),
    radial-gradient(circle at 84% 6%, rgba(9, 182, 216, 0.08), transparent 32%),
    var(--dash-page);
}

html.dark .dashboard-page {
  --dash-page: #061022;
  --dash-card: rgba(8, 21, 43, 0.82);
  --dash-card-strong: rgba(11, 27, 54, 0.92);
  --dash-border: rgba(55, 133, 255, 0.28);
  --dash-title: #f4f8ff;
  --dash-text: #aebfdb;
  --dash-muted: #70809b;
  --dash-blue: #1686ff;
  --dash-cyan: #22d3ee;
  --dash-green: #2ee59d;
  --dash-violet: #8b5cf6;
  --dash-amber: #fbbf24;
  --dash-shadow: 0 18px 56px rgba(0, 0, 0, 0.28);
}

.hero-card,
.metric-card,
.panel {
  border: 1px solid var(--dash-border);
  background: linear-gradient(145deg, var(--dash-card-strong), var(--dash-card));
  box-shadow: var(--dash-shadow);
  backdrop-filter: blur(16px);
}

.hero-card {
  position: relative;
  display: grid;
  grid-template-columns: minmax(260px, 0.9fr) minmax(320px, 1.1fr) minmax(160px, 0.5fr);
  gap: 20px;
  align-items: center;
  min-height: 246px;
  padding: 30px;
  overflow: hidden;
  border-radius: 6px;

  &::before {
    position: absolute;
    inset: 0;
    content: "";
    background:
      linear-gradient(rgba(22, 134, 255, 0.08) 1px, transparent 1px),
      linear-gradient(90deg, rgba(22, 134, 255, 0.08) 1px, transparent 1px);
    background-size: 42px 42px;
    mask-image: linear-gradient(90deg, #000 0%, rgba(0, 0, 0, 0.82) 55%, transparent 100%);
    pointer-events: none;
  }
}

.hero-copy,
.holo-stage,
.hero-badges,
.score-card {
  position: relative;
  z-index: 1;
}

.online-pill {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  color: var(--dash-cyan);
  font-size: 13px;
  font-weight: 700;

  span {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--dash-green);
    box-shadow: 0 0 16px var(--dash-green);
  }
}

.hero-copy {
  h1 {
    margin: 16px 0 8px;
    color: var(--dash-title);
    font-size: clamp(26px, 2.2vw, 34px);
    font-weight: 800;
    letter-spacing: 0;
  }
}

.hero-date {
  margin: 0 0 8px;
  color: var(--dash-text);
  font-size: 14px;
}

.hero-desc {
  max-width: 460px;
  margin: 0;
  color: var(--dash-muted);
  line-height: 1.8;
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 22px;

  :deep(.el-button) {
    height: 38px;
    padding: 0 18px;
    border-radius: 6px;
  }
}

.holo-stage {
  min-height: 210px;
  perspective: 900px;
}

.holo-grid {
  position: absolute;
  inset: 40px 34px 10px;
  border: 1px solid rgba(34, 211, 238, 0.18);
  transform: rotateX(62deg) rotateZ(0deg);
  background:
    linear-gradient(rgba(34, 211, 238, 0.13) 1px, transparent 1px),
    linear-gradient(90deg, rgba(34, 211, 238, 0.13) 1px, transparent 1px);
  background-size: 22px 22px;
  box-shadow: 0 0 54px rgba(22, 134, 255, 0.18);
}

.holo-base {
  position: absolute;
  left: 50%;
  border: 1px solid rgba(34, 211, 238, 0.38);
  background: linear-gradient(180deg, rgba(22, 134, 255, 0.34), rgba(34, 211, 238, 0.08));
  box-shadow: 0 0 34px rgba(22, 134, 255, 0.3);
  transform: translateX(-50%) rotateX(62deg);
}

.base-1 {
  bottom: 36px;
  width: 230px;
  height: 104px;
}

.base-2 {
  bottom: 62px;
  width: 168px;
  height: 82px;
}

.holo-cube {
  position: absolute;
  top: 46px;
  left: 50%;
  width: 108px;
  height: 108px;
  transform: translateX(-50%) rotateX(-18deg) rotateY(35deg);
  transform-style: preserve-3d;
  animation: floatCube 5.6s ease-in-out infinite;

  span {
    position: absolute;
    inset: 0;
    border: 1px solid rgba(34, 211, 238, 0.58);
    background: linear-gradient(135deg, rgba(34, 211, 238, 0.28), rgba(22, 134, 255, 0.15));
    box-shadow: inset 0 0 30px rgba(34, 211, 238, 0.18);
  }

  span:nth-child(1) { transform: translateZ(54px); }
  span:nth-child(2) { transform: rotateY(180deg) translateZ(54px); }
  span:nth-child(3) { transform: rotateY(90deg) translateZ(54px); }
  span:nth-child(4) { transform: rotateY(-90deg) translateZ(54px); }
  span:nth-child(5) { transform: rotateX(90deg) translateZ(54px); }
  span:nth-child(6) { transform: rotateX(-90deg) translateZ(54px); }
}

@keyframes floatCube {
  0%,
  100% {
    transform: translateX(-50%) translateY(0) rotateX(-18deg) rotateY(35deg);
  }

  50% {
    transform: translateX(-50%) translateY(-10px) rotateX(-14deg) rotateY(48deg);
  }
}

.spark {
  position: absolute;
  width: 14px;
  height: 14px;
  border: 1px solid rgba(34, 211, 238, 0.52);
  background: rgba(22, 134, 255, 0.42);
  box-shadow: 0 0 18px rgba(34, 211, 238, 0.48);
  transform: rotate(45deg);
  animation: sparkPulse 3.4s ease-in-out infinite;
}

.spark-1 { top: 45px; left: 13%; }
.spark-2 { top: 80px; left: 27%; animation-delay: .2s; }
.spark-3 { top: 120px; left: 18%; animation-delay: .4s; }
.spark-4 { top: 30px; right: 20%; animation-delay: .6s; }
.spark-5 { top: 88px; right: 12%; animation-delay: .8s; }
.spark-6 { bottom: 42px; right: 25%; animation-delay: 1s; }
.spark-7 { bottom: 28px; left: 36%; animation-delay: 1.2s; }
.spark-8 { top: 22px; left: 50%; animation-delay: 1.4s; }

@keyframes sparkPulse {
  50% {
    transform: translateY(-10px) rotate(45deg) scale(1.08);
    opacity: 0.65;
  }
}

.hero-badges {
  position: absolute;
  inset: 30px 260px 30px auto;
  width: 520px;
  pointer-events: none;
}

.float-badge {
  position: absolute;
  display: grid;
  grid-template-columns: 38px 1fr;
  gap: 2px 10px;
  min-width: 132px;
  padding: 12px;
  border: 1px solid rgba(67, 136, 231, 0.2);
  border-radius: 8px;
  background: rgba(7, 19, 40, 0.54);
  animation: badgeIn .42s ease both;
  animation-delay: var(--delay);

  span {
    color: var(--dash-text);
    font-size: 12px;
  }

  strong {
    color: var(--dash-title);
    font-size: 22px;
    line-height: 1;
  }
}

html:not(.dark) .float-badge {
  background: rgba(255, 255, 255, 0.62);
}

.float-badge:nth-child(1) { top: 0; left: 40px; }
.float-badge:nth-child(2) { top: 12px; right: 0; }
.float-badge:nth-child(3) { right: -12px; bottom: 18px; }
.float-badge:nth-child(4) { left: 0; bottom: 38px; }

@keyframes badgeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
}

.float-icon,
.metric-icon {
  display: grid;
  place-items: center;
  border-radius: 8px;
  color: var(--tone);
  background: color-mix(in srgb, var(--tone) 16%, transparent);
}

.float-icon {
  grid-row: span 2;
  width: 38px;
  height: 38px;
  font-size: 20px;
}

.score-card {
  justify-self: end;
  display: grid;
  justify-items: center;
  align-self: center;
  width: 196px;
  min-width: 196px;
  padding-top: 4px;
}

.score-chart {
  width: 190px;
  height: 178px;
}

.score-card span {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 56px;
  height: 24px;
  margin-top: -4px;
  color: var(--dash-text);
  font-size: 13px;
  line-height: 1;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.metric-card {
  position: relative;
  display: grid;
  grid-template-columns: 58px 1fr 118px;
  gap: 12px;
  align-items: center;
  min-height: 106px;
  padding: 18px;
  overflow: hidden;
  border-radius: 6px;

  em {
    position: absolute;
    top: 16px;
    right: 16px;
    color: var(--dash-green);
    font-style: normal;
    font-size: 12px;
  }
}

.metric-icon {
  width: 52px;
  height: 52px;
  font-size: 28px;
}

.metric-body {
  display: grid;
  gap: 2px;

  span,
  small {
    color: var(--dash-muted);
    font-size: 12px;
  }

  strong {
    color: var(--dash-title);
    font-size: 26px;
    line-height: 1.05;
  }
}

.sparkline {
  width: 120px;
  height: 44px;
}

.tone-cyan { --tone: var(--dash-cyan); }
.tone-green { --tone: var(--dash-green); }
.tone-violet { --tone: var(--dash-violet); }
.tone-amber { --tone: var(--dash-amber); }

.content-grid {
  display: grid;
  grid-template-columns: 1.2fr 1fr 1.05fr;
  gap: 12px;
}

.panel {
  min-width: 0;
  padding: 18px;
  border-radius: 6px;

  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  h2 {
    margin: 0;
    color: var(--dash-title);
    font-size: 18px;
  }

  p {
    margin: 6px 0 0;
    color: var(--dash-muted);
  }

  button {
    padding: 0;
    color: var(--dash-blue);
    border: 0;
    background: transparent;
    cursor: pointer;
  }
}

.chart-main {
  width: 100%;
  height: 260px;
}

.resource-panel .chart-main {
  height: 320px;
}

.source-layout {
  display: flex;
  gap: 18px;
  align-items: center;
  min-height: 260px;
}

.source-chart {
  flex: 0 0 46%;
  width: 100%;
  min-width: 190px;
  height: 236px;
}

.source-list {
  display: grid;
  flex: 1 1 0;
  gap: 14px;
  min-width: 0;
  overflow: hidden;

  div {
    display: grid;
    grid-template-columns: 12px minmax(0, 1fr) minmax(28px, auto) minmax(50px, auto);
    gap: 8px;
    align-items: center;
    min-width: 0;
    color: var(--dash-text);
    font-size: 13px;
  }

  i {
    width: 9px;
    height: 9px;
    border: 2px solid var(--source-color);
    border-radius: 3px;
    box-shadow: 0 0 12px color-mix(in srgb, var(--source-color) 48%, transparent);
  }

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    color: var(--dash-title);
    font-weight: 800;
  }

  em {
    overflow: hidden;
    color: var(--dash-muted);
    font-style: normal;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.source-blue { --source-color: var(--dash-blue); }
.source-violet { --source-color: var(--dash-violet); }
.source-green { --source-color: var(--dash-green); }
.source-amber { --source-color: var(--dash-amber); }
.source-cyan { --source-color: var(--dash-cyan); }

.message-funnel {
  display: grid;
  grid-template-columns: minmax(150px, 0.82fr) minmax(0, 1fr);
  gap: 16px;
  align-items: center;
  min-height: 216px;
  overflow: hidden;
}

.message-chart {
  width: 100%;
  min-width: 0;
  height: 214px;
}

.message-legend {
  display: grid;
  gap: 14px;
  min-width: 0;

  div {
    display: grid;
    grid-template-columns: 12px minmax(0, 1fr) minmax(28px, auto) minmax(54px, auto);
    gap: 8px;
    align-items: center;
    min-width: 0;
    color: var(--dash-text);
    font-size: 14px;
  }

  i {
    width: 8px;
    height: 8px;
    border: 2px solid var(--message-color);
    border-radius: 3px;
    box-shadow: 0 0 12px color-mix(in srgb, var(--message-color) 58%, transparent);
  }

  span {
    overflow: hidden;
    color: var(--dash-text);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    color: var(--dash-title);
    font-size: 16px;
    font-weight: 800;
    text-align: right;
  }

  em {
    overflow: hidden;
    color: var(--dash-muted);
    font-style: normal;
    text-align: right;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.message-blue { --message-color: var(--dash-blue); }
.message-cyan { --message-color: var(--dash-cyan); }
.message-amber { --message-color: var(--dash-amber); }
.message-violet { --message-color: var(--dash-violet); }

.logs-panel,
.message-panel,
.activity-panel {
  min-height: 250px;
}

.table-wrap {
  overflow-x: auto;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;

  th,
  td {
    padding: 12px 10px;
    text-align: left;
    white-space: nowrap;
    border-bottom: 1px solid color-mix(in srgb, var(--dash-border) 70%, transparent);
  }

  th {
    color: var(--dash-text);
    font-weight: 600;
    background: color-mix(in srgb, var(--dash-blue) 8%, transparent);
  }

  td {
    color: var(--dash-text);
  }
}

.success-tag {
  color: var(--dash-green);
}

.activity-list {
  display: grid;
  gap: 8px;
}

.activity-item {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid color-mix(in srgb, var(--dash-border) 58%, transparent);

  strong {
    display: block;
    overflow: hidden;
    color: var(--dash-blue);
    font-size: 14px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  p {
    overflow: hidden;
    margin: 4px 0 0;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  time {
    color: var(--dash-muted);
    font-size: 12px;
    white-space: nowrap;
  }
}

.activity-icon {
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  color: #fff;
  border-radius: 8px;
}

.act-blue { background: linear-gradient(135deg, #1677ff, #1d4ed8); }
.act-cyan { background: linear-gradient(135deg, #22d3ee, #0891b2); }
.act-amber { background: linear-gradient(135deg, #fbbf24, #b7791f); }
.act-violet { background: linear-gradient(135deg, #8b5cf6, #6d28d9); }

@media (max-width: 1680px) {
  .hero-card {
    grid-template-columns: minmax(260px, 1fr) minmax(280px, 0.8fr) 170px;
  }

  .content-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .hero-badges {
    display: none;
  }
}

@media (max-width: 1400px) {
  .message-funnel {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .message-chart {
    height: 190px;
  }

  .metric-card {
    grid-template-columns: 52px 1fr;
  }

  .sparkline {
    display: none;
  }
}

@media (max-width: 1180px) {
  .hero-card,
  .content-grid {
    grid-template-columns: 1fr 1fr;
  }

  .score-card {
    display: none;
  }

  .metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 1040px) {
  .content-grid {
    grid-template-columns: 1fr;
  }

  .source-layout {
    align-items: stretch;
    flex-direction: column;
  }

  .source-chart {
    flex-basis: auto;
    min-width: 0;
  }
}

@media (max-width: 780px) {
  .dashboard-page {
    padding: 14px;
  }

  .hero-card,
  .content-grid,
  .metric-grid {
    grid-template-columns: 1fr;
  }

  .holo-stage {
    min-height: 180px;
    order: -1;
  }

  .hero-card {
    padding: 20px;
  }

  .message-funnel {
    grid-template-columns: 1fr;
  }

  .message-legend div {
    grid-template-columns: 12px minmax(0, 1fr) auto;
  }

  .message-legend em {
    grid-column: 2 / -1;
    text-align: left;
  }
}
</style>
