<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  getSiteOperationsDashboard,
  type SiteOperationsDashboard
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "SiteAdminOperations" });

const loading = ref(false);
const dashboard = ref<SiteOperationsDashboard>();

const conversion = computed(() =>
  `${((dashboard.value?.conversion_rate ?? 0) * 100).toFixed(2)}%`
);

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getSiteOperationsDashboard();
    dashboard.value = res.dashboard;
  } catch {
    message("官网运营数据加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="ops-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Site operations</p>
        <h2>官网运营仪表盘</h2>
        <p>把访问、内容、项目、留言放到一个页面，方便判断今天该补内容还是处理线索。</p>
      </div>
      <el-button @click="loadData">刷新</el-button>
    </section>

    <section class="metric-grid">
      <div class="metric-card">
        <span>总访问</span>
        <strong>{{ dashboard?.analytics.visit_count ?? 0 }}</strong>
        <small>近 24 小时 {{ dashboard?.analytics.today_visits ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>已发布文章</span>
        <strong>{{ dashboard?.analytics.article_count ?? 0 }}</strong>
        <small>草稿 {{ dashboard?.draft_resources ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>项目作品</span>
        <strong>{{ dashboard?.published_projects ?? 0 }}</strong>
        <small>精选 {{ dashboard?.featured_projects ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>留言转化</span>
        <strong>{{ conversion }}</strong>
        <small>待处理 {{ dashboard?.analytics.pending_messages ?? 0 }}</small>
      </div>
    </section>

    <section class="ops-layout">
      <div class="panel wide">
        <div class="panel-title">
          <h3>访问趋势</h3>
          <span>最近 7 天</span>
        </div>
        <div class="trend-list">
          <div v-for="item in dashboard?.analytics.visits_by_day ?? []" :key="item.date">
            <span>{{ item.date }}</span>
            <el-progress :percentage="Math.min(100, item.visits * 10)" :show-text="false" />
            <strong>{{ item.visits }}</strong>
          </div>
          <el-empty v-if="!dashboard?.analytics.visits_by_day?.length" description="暂无访问趋势" />
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>内容健康</h3>
          <span>需要处理的队列</span>
        </div>
        <div class="health-list">
          <div v-for="item in dashboard?.content_health ?? []" :key="item.label">
            <span>{{ item.label }}</span>
            <el-tag :type="item.tone === 'success' ? 'success' : 'warning'" effect="light">
              {{ item.value }}
            </el-tag>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>热门页面</h3>
          <span>按访问排序</span>
        </div>
        <div class="rank-list">
          <div v-for="item in dashboard?.analytics.top_pages ?? []" :key="item.path">
            <span>{{ item.path }}</span>
            <strong>{{ item.visits }}</strong>
          </div>
          <el-empty v-if="!dashboard?.analytics.top_pages?.length" description="暂无页面数据" />
        </div>
      </div>

      <div class="panel wide">
        <div class="panel-title">
          <h3>项目作品</h3>
          <span>发布与精选优先</span>
        </div>
        <el-table :data="dashboard?.top_projects ?? []" stripe>
          <el-table-column prop="name" label="项目" min-width="180" show-overflow-tooltip />
          <el-table-column prop="role" label="角色" width="130" show-overflow-tooltip />
          <el-table-column prop="stack_tags" label="技术栈" min-width="180" show-overflow-tooltip />
          <el-table-column prop="priority" label="优先级" width="90" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="row.status === 'published' ? 'success' : 'info'" effect="light">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>设备分布</h3>
          <span>访问设备</span>
        </div>
        <div class="rank-list">
          <div v-for="item in dashboard?.analytics.device_stats ?? []" :key="item.device">
            <span>{{ item.device }}</span>
            <strong>{{ item.visits }}</strong>
          </div>
        </div>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>最新留言</h3>
          <span>{{ dayjs(dashboard?.generated_at).format("MM-DD HH:mm") }}</span>
        </div>
        <div class="message-list">
          <div v-for="item in dashboard?.recent_messages ?? []" :key="item.id">
            <strong>{{ item.visitor_name || "访客" }}</strong>
            <p>{{ item.content }}</p>
            <span>{{ item.status }} · {{ dayjs(item.created_at).format("MM-DD HH:mm") }}</span>
          </div>
          <el-empty v-if="!dashboard?.recent_messages?.length" description="暂无留言" />
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.ops-page {
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

.page-head p:last-child,
.panel-title span,
.metric-card span,
.metric-card small,
.message-list span {
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

.metric-card strong {
  font-size: 26px;
}

.ops-layout {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.panel {
  min-width: 0;
  padding: 18px;
}

.panel.wide {
  grid-column: span 2;
}

.panel-title,
.trend-list > div,
.health-list > div,
.rank-list > div {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.trend-list,
.health-list,
.rank-list,
.message-list {
  display: grid;
  gap: 12px;
  margin-top: 14px;
}

.trend-list .el-progress {
  flex: 1;
}

.rank-list span,
.message-list p {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.message-list > div {
  display: grid;
  gap: 6px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.message-list p {
  margin: 0;
}

@media (max-width: 1100px) {
  .metric-grid,
  .ops-layout {
    grid-template-columns: 1fr 1fr;
  }

  .panel.wide {
    grid-column: span 2;
  }
}
</style>
