<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  deleteSiteAnnouncement,
  deleteSiteBanner,
  deleteSiteMessage,
  deleteSiteProject,
  deleteSiteResource,
  deleteSiteTechStack,
  deleteSiteTimelineEvent,
  getSiteAnalytics,
  getSiteAnnouncements,
  getSiteBanners,
  getSiteMessages,
  getSiteProjects,
  getSiteResources,
  getSiteTechStacks,
  getSiteTimelineEvents,
  saveSiteAnnouncement,
  saveSiteBanner,
  saveSiteMessage,
  saveSiteProject,
  saveSiteResource,
  saveSiteTechStack,
  saveSiteTimelineEvent,
  uploadSiteAsset,
  type SiteAnalytics,
  type SiteAnnouncement,
  type SiteBanner,
  type SiteMessage,
  type SiteProject,
  type SiteResource,
  type SiteTechStack,
  type SiteTimelineEvent
} from "@/api/admin";
import { message } from "@/utils/message";
import { hasAuth } from "@/router/utils";

defineOptions({ name: "SiteAdminContent" });

type TabKey =
  | "analytics"
  | "announcements"
  | "banners"
  | "resources"
  | "projects"
  | "timeline"
  | "messages"
  | "tech";

const activeTab = ref<TabKey>("analytics");
const loading = ref(false);
const dialogVisible = ref(false);
const dialogMode = ref<"create" | "edit">("create");
const editingId = ref<number>();
const uploadTarget = ref<"image_url" | "cover_url" | "icon_url">("image_url");

const announcements = ref<SiteAnnouncement[]>([]);
const banners = ref<SiteBanner[]>([]);
const resources = ref<SiteResource[]>([]);
const projects = ref<SiteProject[]>([]);
const timeline = ref<SiteTimelineEvent[]>([]);
const messages = ref<SiteMessage[]>([]);
const techStacks = ref<SiteTechStack[]>([]);
const analytics = ref<SiteAnalytics>();
const totals = reactive<Record<TabKey, number>>({
  analytics: 0,
  announcements: 0,
  banners: 0,
  resources: 0,
  projects: 0,
  timeline: 0,
  messages: 0,
  tech: 0
});
const pagination = reactive({ page: 1, pageSize: 10 });
const statusFilter = ref("");

const canWrite = computed(() => hasAuth("site:write"));
const isTableTab = computed(() => activeTab.value !== "analytics");

const form = reactive<any>({
  title: "",
  slug: "",
  content: "",
  markdown_content: "",
  link_url: "",
  is_active: true,
  sort_order: 10,
  starts_at: "",
  ends_at: "",
  subtitle: "",
  image_url: "",
  summary: "",
  category: "frontend",
  cover_url: "",
  tags: "",
  seo_title: "",
  seo_description: "",
  seo_keywords: "",
  status: "published",
  is_featured: false,
  published_at: "",
  demo_url: "",
  repo_url: "",
  stack_tags: "",
  phase: "",
  event_type: "learning",
  happened_at: "",
  name: "",
  level: 70,
  icon_url: "",
  description: "",
  visitor_name: "",
  email: "",
  reply: "",
  is_public: true
});

const currentRows = computed(() => {
  if (activeTab.value === "announcements") return announcements.value;
  if (activeTab.value === "banners") return banners.value;
  if (activeTab.value === "resources") return resources.value;
  if (activeTab.value === "projects") return projects.value;
  if (activeTab.value === "timeline") return timeline.value;
  if (activeTab.value === "messages") return messages.value;
  if (activeTab.value === "tech") return techStacks.value;
  return [];
});

const pageTitle = computed(() => {
  const map: Record<TabKey, string> = {
    analytics: "访问统计",
    announcements: "公告",
    banners: "轮播图",
    resources: "博客文章",
    projects: "项目 Demo",
    timeline: "时间轴",
    messages: "留言",
    tech: "技术栈"
  };
  return map[activeTab.value];
});

const resetForm = () => {
  Object.assign(form, {
    title: "",
    slug: "",
    content: "",
    markdown_content: "",
    link_url: "",
    is_active: true,
    sort_order: 10,
    starts_at: "",
    ends_at: "",
    subtitle: "",
    image_url: "",
    summary: "",
    category: "frontend",
    cover_url: "",
    tags: "",
    seo_title: "",
    seo_description: "",
    seo_keywords: "",
    status: activeTab.value === "messages" ? "pending" : "published",
    is_featured: false,
    published_at: "",
    demo_url: "",
    repo_url: "",
    stack_tags: "",
    phase: "",
    event_type: "learning",
    happened_at: "",
    name: "",
    level: 70,
    icon_url: "",
    description: "",
    visitor_name: "",
    email: "",
    reply: "",
    is_public: true
  });
};

const fillForm = (row: any) => {
  resetForm();
  Object.keys(form).forEach(key => {
    if (row[key] !== undefined) form[key] = row[key] ?? "";
  });
};

const normalizePayload = () => {
  const payload = { ...form };
  ["starts_at", "ends_at", "published_at", "happened_at"].forEach(key => {
    if (!payload[key]) payload[key] = null;
  });
  return payload;
};

const loadData = async () => {
  loading.value = true;
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      status: statusFilter.value || undefined
    };
    if (activeTab.value === "analytics") {
      const res = await getSiteAnalytics();
      analytics.value = res.analytics;
    } else if (activeTab.value === "announcements") {
      const res = await getSiteAnnouncements(params);
      announcements.value = res.announcements ?? [];
      totals.announcements = res.total ?? 0;
    } else if (activeTab.value === "banners") {
      const res = await getSiteBanners(params);
      banners.value = res.banners ?? [];
      totals.banners = res.total ?? 0;
    } else if (activeTab.value === "resources") {
      const res = await getSiteResources(params);
      resources.value = res.resources ?? [];
      totals.resources = res.total ?? 0;
    } else if (activeTab.value === "projects") {
      const res = await getSiteProjects(params);
      projects.value = res.projects ?? [];
      totals.projects = res.total ?? 0;
    } else if (activeTab.value === "timeline") {
      const res = await getSiteTimelineEvents(params);
      timeline.value = res.timeline ?? [];
      totals.timeline = res.total ?? 0;
    } else if (activeTab.value === "messages") {
      const res = await getSiteMessages(params);
      messages.value = res.messages ?? [];
      totals.messages = res.total ?? 0;
    } else {
      const res = await getSiteTechStacks(params);
      techStacks.value = res.tech_stacks ?? [];
      totals.tech = res.total ?? 0;
    }
  } catch {
    message("官网数据加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const handleTabChange = () => {
  pagination.page = 1;
  statusFilter.value = "";
  loadData();
};

const openCreate = () => {
  resetForm();
  editingId.value = undefined;
  dialogMode.value = "create";
  dialogVisible.value = true;
};

const openEdit = (row: any) => {
  fillForm(row);
  editingId.value = row.id;
  dialogMode.value = "edit";
  dialogVisible.value = true;
};

const saveCurrent = async () => {
  const payload = normalizePayload();
  if (activeTab.value === "announcements") {
    await saveSiteAnnouncement(payload, editingId.value);
  } else if (activeTab.value === "banners") {
    await saveSiteBanner(payload, editingId.value);
  } else if (activeTab.value === "resources") {
    await saveSiteResource(payload, editingId.value);
  } else if (activeTab.value === "projects") {
    await saveSiteProject(payload, editingId.value);
  } else if (activeTab.value === "timeline") {
    await saveSiteTimelineEvent(payload, editingId.value);
  } else if (activeTab.value === "messages" && editingId.value) {
    await saveSiteMessage(payload, editingId.value);
  } else {
    await saveSiteTechStack(payload, editingId.value);
  }
  message("已保存", { type: "success" });
  dialogVisible.value = false;
  await loadData();
};

const removeCurrent = async (row: any) => {
  if (activeTab.value === "announcements") await deleteSiteAnnouncement(row.id);
  else if (activeTab.value === "banners") await deleteSiteBanner(row.id);
  else if (activeTab.value === "resources") await deleteSiteResource(row.id);
  else if (activeTab.value === "projects") await deleteSiteProject(row.id);
  else if (activeTab.value === "timeline") await deleteSiteTimelineEvent(row.id);
  else if (activeTab.value === "messages") await deleteSiteMessage(row.id);
  else await deleteSiteTechStack(row.id);
  message("已删除", { type: "success" });
  await loadData();
};

const beforeUpload = async (file: File) => {
  try {
    const res = await uploadSiteAsset(file);
    form[uploadTarget.value] = res.url;
    message("上传完成", { type: "success" });
  } catch {
    message("上传失败", { type: "error" });
  }
  return false;
};

const formatTime = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";

const initTabFromQuery = () => {
  const tab = new URLSearchParams(window.location.search).get("tab") as TabKey;
  if (
    [
      "analytics",
      "announcements",
      "banners",
      "resources",
      "projects",
      "timeline",
      "messages",
      "tech"
    ].includes(tab)
  ) {
    activeTab.value = tab;
  }
};

onMounted(() => {
  initTabFromQuery();
  loadData();
});
</script>

<template>
  <div class="site-page">
    <div class="page-head">
      <div>
        <h2>官网管理</h2>
        <p>维护博客文章、知识库内容、项目展示、留言审核和访问分析。</p>
      </div>
      <Auth v-if="activeTab !== 'analytics' && activeTab !== 'messages'" value="site:write">
        <el-button type="primary" @click="openCreate">新增{{ pageTitle }}</el-button>
      </Auth>
    </div>

    <div class="toolbar">
      <el-tabs v-model="activeTab" class="content-tabs" @tab-change="handleTabChange">
        <el-tab-pane label="访问统计" name="analytics" />
        <el-tab-pane label="博客文章" name="resources" />
        <el-tab-pane label="留言审核" name="messages" />
        <el-tab-pane label="时间轴实验室" name="timeline" />
        <el-tab-pane label="公告" name="announcements" />
        <el-tab-pane label="轮播图" name="banners" />
        <el-tab-pane label="项目 Demo" name="projects" />
        <el-tab-pane label="技术栈" name="tech" />
      </el-tabs>
      <div class="toolbar-actions">
        <el-select v-if="isTableTab" v-model="statusFilter" clearable placeholder="状态" style="width: 140px" @change="loadData">
          <el-option v-if="activeTab !== 'resources' && activeTab !== 'projects' && activeTab !== 'timeline' && activeTab !== 'messages'" label="启用" value="active" />
          <el-option v-if="activeTab !== 'resources' && activeTab !== 'projects' && activeTab !== 'timeline' && activeTab !== 'messages'" label="停用" value="inactive" />
          <el-option v-if="activeTab === 'resources' || activeTab === 'projects' || activeTab === 'timeline'" label="已发布" value="published" />
          <el-option v-if="activeTab === 'resources' || activeTab === 'projects' || activeTab === 'timeline'" label="草稿" value="draft" />
          <el-option v-if="activeTab === 'messages'" label="待审核" value="pending" />
          <el-option v-if="activeTab === 'messages'" label="已通过" value="approved" />
          <el-option v-if="activeTab === 'messages'" label="已拒绝" value="rejected" />
        </el-select>
        <el-button @click="loadData">刷新</el-button>
      </div>
    </div>

    <div v-if="activeTab === 'analytics'" v-loading="loading" class="analytics-grid">
      <div class="metric-card"><span>总访问量</span><strong>{{ analytics?.visit_count ?? 0 }}</strong></div>
      <div class="metric-card"><span>近 24 小时</span><strong>{{ analytics?.today_visits ?? 0 }}</strong></div>
      <div class="metric-card"><span>已发布文章</span><strong>{{ analytics?.article_count ?? 0 }}</strong></div>
      <div class="metric-card"><span>待审核留言</span><strong>{{ analytics?.pending_messages ?? 0 }}</strong></div>
      <div class="panel-card span-2">
        <h3>最近 7 天访问</h3>
        <div class="trend-list">
          <div v-for="item in analytics?.visits_by_day" :key="item.date">
            <span>{{ item.date }}</span>
            <el-progress :percentage="Math.min(item.visits * 10, 100)" :format="() => `${item.visits}`" />
          </div>
        </div>
      </div>
      <div class="panel-card">
        <h3>热门路径</h3>
        <p v-for="item in analytics?.top_pages" :key="item.path">
          <span>{{ item.path }}</span><strong>{{ item.visits }}</strong>
        </p>
      </div>
      <div class="panel-card">
        <h3>设备类型</h3>
        <p v-for="item in analytics?.device_stats" :key="item.device">
          <span>{{ item.device }}</span><strong>{{ item.visits }}</strong>
        </p>
      </div>
      <div class="panel-card span-2">
        <h3>热门文章</h3>
        <p v-for="item in analytics?.top_articles" :key="item.id">
          <span>{{ item.title }}</span><strong>{{ item.view_count }}</strong>
        </p>
      </div>
    </div>

    <template v-else>
      <el-table v-loading="loading" :data="currentRows" class="site-table">
        <el-table-column v-if="activeTab !== 'messages'" prop="sort_order" label="排序" width="80" />
        <el-table-column v-if="activeTab === 'tech'" prop="name" label="名称" min-width="140" />
        <el-table-column v-else-if="activeTab === 'projects'" prop="name" label="项目" min-width="180" />
        <el-table-column v-else-if="activeTab === 'messages'" prop="visitor_name" label="访客" width="140" />
        <el-table-column v-else prop="title" label="标题" min-width="180" />
        <el-table-column v-if="activeTab === 'resources'" prop="slug" label="Slug" min-width="160" />
        <el-table-column v-if="activeTab === 'banners'" prop="subtitle" label="副标题" min-width="220" />
        <el-table-column v-if="activeTab === 'timeline'" prop="phase" label="阶段" width="120" />
        <el-table-column v-if="activeTab === 'timeline'" prop="event_type" label="类型" width="120" />
        <el-table-column v-if="activeTab === 'resources' || activeTab === 'projects' || activeTab === 'timeline'" prop="summary" label="摘要" min-width="260" />
        <el-table-column v-if="activeTab === 'messages'" prop="content" label="留言内容" min-width="260" />
        <el-table-column v-if="activeTab === 'messages'" prop="reply" label="回复" min-width="220" />
        <el-table-column v-if="activeTab === 'projects'" prop="stack_tags" label="技术标签" min-width="220" />
        <el-table-column v-if="activeTab === 'timeline'" prop="tags" label="标签" min-width="180" />
        <el-table-column v-if="activeTab === 'tech' || activeTab === 'resources'" prop="category" label="分类" width="120" />
        <el-table-column v-if="activeTab === 'tech'" prop="level" label="掌握度" width="120">
          <template #default="{ row }">
            <el-progress :percentage="row.level" :stroke-width="8" />
          </template>
        </el-table-column>
        <el-table-column v-if="activeTab === 'resources'" prop="view_count" label="阅读" width="90" />
        <el-table-column v-if="activeTab === 'timeline'" label="发生时间" width="170">
          <template #default="{ row }">{{ formatTime(row.happened_at) }}</template>
        </el-table-column>
        <el-table-column v-if="activeTab === 'resources' || activeTab === 'projects' || activeTab === 'timeline' || activeTab === 'messages'" prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'published' || row.status === 'approved' ? 'success' : row.status === 'rejected' ? 'danger' : 'info'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column v-if="activeTab !== 'resources' && activeTab !== 'projects' && activeTab !== 'timeline' && activeTab !== 'messages'" prop="is_active" label="启用" width="90">
          <template #default="{ row }">
            <el-tag :type="row.is_active ? 'success' : 'info'">{{ row.is_active ? "启用" : "停用" }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" fixed="right" width="150">
          <template #default="{ row }">
            <Auth value="site:write">
              <el-button link type="primary" @click="openEdit(row)">{{ activeTab === "messages" ? "处理" : "编辑" }}</el-button>
              <el-popconfirm title="确认删除？" @confirm="removeCurrent(row)">
                <template #reference>
                  <el-button link type="danger">删除</el-button>
                </template>
              </el-popconfirm>
            </Auth>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="totals[activeTab]"
          layout="total, sizes, prev, pager, next"
          @change="loadData"
        />
      </div>
    </template>

    <el-dialog v-model="dialogVisible" :title="`${dialogMode === 'create' ? '新增' : '编辑'}${pageTitle}`" width="820px">
      <el-form :model="form" label-width="104px" class="site-form">
        <template v-if="activeTab === 'announcements'">
          <el-form-item label="标题"><el-input v-model="form.title" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="form.content" type="textarea" :rows="4" /></el-form-item>
          <el-form-item label="链接"><el-input v-model="form.link_url" /></el-form-item>
          <el-form-item label="展示时间">
            <el-date-picker v-model="form.starts_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" placeholder="开始时间" />
            <el-date-picker v-model="form.ends_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" placeholder="结束时间" />
          </el-form-item>
        </template>

        <template v-if="activeTab === 'banners'">
          <el-form-item label="标题"><el-input v-model="form.title" /></el-form-item>
          <el-form-item label="副标题"><el-input v-model="form.subtitle" /></el-form-item>
          <el-form-item label="图片">
            <el-input v-model="form.image_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="file => { uploadTarget = 'image_url'; return beforeUpload(file); }">
                  <el-button>上传</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="链接"><el-input v-model="form.link_url" /></el-form-item>
        </template>

        <template v-if="activeTab === 'resources'">
          <el-form-item label="标题"><el-input v-model="form.title" /></el-form-item>
          <el-form-item label="SEO Slug"><el-input v-model="form.slug" placeholder="react-hooks-notes" /></el-form-item>
          <el-form-item label="摘要"><el-input v-model="form.summary" type="textarea" :rows="3" /></el-form-item>
          <el-form-item label="Markdown"><el-input v-model="form.markdown_content" type="textarea" :rows="9" /></el-form-item>
          <el-form-item label="渲染正文"><el-input v-model="form.content" type="textarea" :rows="5" /></el-form-item>
          <el-form-item label="分类">
            <el-select v-model="form.category">
              <el-option label="前端" value="frontend" />
              <el-option label="后端" value="backend" />
              <el-option label="数据库" value="database" />
              <el-option label="学习笔记" value="learning" />
              <el-option label="项目复盘" value="project" />
            </el-select>
          </el-form-item>
          <el-form-item label="封面">
            <el-input v-model="form.cover_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="file => { uploadTarget = 'cover_url'; return beforeUpload(file); }">
                  <el-button>上传</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="标签"><el-input v-model="form.tags" placeholder="React,Go,PostgreSQL" /></el-form-item>
          <el-form-item label="外链"><el-input v-model="form.link_url" /></el-form-item>
          <el-form-item label="SEO 标题"><el-input v-model="form.seo_title" /></el-form-item>
          <el-form-item label="SEO 描述"><el-input v-model="form.seo_description" type="textarea" :rows="2" /></el-form-item>
          <el-form-item label="SEO 关键词"><el-input v-model="form.seo_keywords" placeholder="React,Hooks,Vite" /></el-form-item>
          <el-form-item label="发布状态">
            <el-segmented v-model="form.status" :options="[{ label: '已发布', value: 'published' }, { label: '草稿', value: 'draft' }]" />
          </el-form-item>
          <el-form-item label="精选"><el-switch v-model="form.is_featured" /></el-form-item>
        </template>

        <template v-if="activeTab === 'projects'">
          <el-form-item label="项目名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="摘要"><el-input v-model="form.summary" type="textarea" :rows="3" /></el-form-item>
          <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="5" /></el-form-item>
          <el-form-item label="封面">
            <el-input v-model="form.cover_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="file => { uploadTarget = 'cover_url'; return beforeUpload(file); }">
                  <el-button>上传</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="演示地址"><el-input v-model="form.demo_url" /></el-form-item>
          <el-form-item label="仓库地址"><el-input v-model="form.repo_url" /></el-form-item>
          <el-form-item label="技术标签"><el-input v-model="form.stack_tags" placeholder="React,Three.js,Go Gin" /></el-form-item>
          <el-form-item label="发布状态">
            <el-segmented v-model="form.status" :options="[{ label: '已发布', value: 'published' }, { label: '草稿', value: 'draft' }]" />
          </el-form-item>
          <el-form-item label="精选"><el-switch v-model="form.is_featured" /></el-form-item>
        </template>

        <template v-if="activeTab === 'timeline'">
          <el-form-item label="标题"><el-input v-model="form.title" /></el-form-item>
          <el-form-item label="摘要"><el-input v-model="form.summary" type="textarea" :rows="3" /></el-form-item>
          <el-form-item label="内容"><el-input v-model="form.content" type="textarea" :rows="5" /></el-form-item>
          <el-form-item label="阶段"><el-input v-model="form.phase" placeholder="Foundation / Growth / Release" /></el-form-item>
          <el-form-item label="类型">
            <el-select v-model="form.event_type">
              <el-option label="学习" value="learning" />
              <el-option label="项目" value="project" />
              <el-option label="发布" value="release" />
              <el-option label="踩坑" value="pitfall" />
              <el-option label="复盘" value="review" />
            </el-select>
          </el-form-item>
          <el-form-item label="标签"><el-input v-model="form.tags" placeholder="React,Go,RBAC" /></el-form-item>
          <el-form-item label="关联链接"><el-input v-model="form.link_url" /></el-form-item>
          <el-form-item label="发生时间">
            <el-date-picker v-model="form.happened_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" placeholder="选择时间" />
          </el-form-item>
          <el-form-item label="发布状态">
            <el-segmented v-model="form.status" :options="[{ label: '已发布', value: 'published' }, { label: '草稿', value: 'draft' }]" />
          </el-form-item>
          <el-form-item label="精选"><el-switch v-model="form.is_featured" /></el-form-item>
        </template>

        <template v-if="activeTab === 'messages'">
          <el-form-item label="访客"><el-input v-model="form.visitor_name" /></el-form-item>
          <el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item>
          <el-form-item label="留言"><el-input v-model="form.content" type="textarea" :rows="4" /></el-form-item>
          <el-form-item label="回复"><el-input v-model="form.reply" type="textarea" :rows="4" /></el-form-item>
          <el-form-item label="审核状态">
            <el-segmented v-model="form.status" :options="[{ label: '待审核', value: 'pending' }, { label: '通过', value: 'approved' }, { label: '拒绝', value: 'rejected' }]" />
          </el-form-item>
          <el-form-item label="公开展示"><el-switch v-model="form.is_public" /></el-form-item>
        </template>

        <template v-if="activeTab === 'tech'">
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="分类"><el-input v-model="form.category" /></el-form-item>
          <el-form-item label="掌握度"><el-slider v-model="form.level" :min="0" :max="100" /></el-form-item>
          <el-form-item label="图标">
            <el-input v-model="form.icon_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="file => { uploadTarget = 'icon_url'; return beforeUpload(file); }">
                  <el-button>上传</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        </template>

        <el-form-item v-if="activeTab !== 'resources' && activeTab !== 'projects' && activeTab !== 'timeline' && activeTab !== 'messages'" label="启用">
          <el-switch v-model="form.is_active" />
        </el-form-item>
        <el-form-item v-if="activeTab !== 'messages'" label="排序"><el-input-number v-model="form.sort_order" :min="0" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!canWrite" @click="saveCurrent">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.site-page {
  padding: 24px;
}

.page-head,
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.page-head h2 {
  margin: 0;
  font-size: 20px;
}

.page-head p {
  margin: 6px 0 0;
  color: var(--app-text-secondary);
}

.toolbar,
.metric-card,
.panel-card {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.toolbar {
  padding: 0 16px;
}

.content-tabs {
  min-width: 520px;
}

.toolbar-actions {
  display: flex;
  gap: 10px;
}

.analytics-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card,
.panel-card {
  padding: 18px;
}

.metric-card span {
  color: var(--app-text-secondary);
  font-size: 13px;
}

.metric-card strong {
  display: block;
  margin-top: 10px;
  font-size: 28px;
}

.panel-card h3 {
  margin: 0 0 14px;
  font-size: 16px;
}

.panel-card p {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin: 10px 0;
  color: var(--app-text-secondary);
}

.panel-card p span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.span-2 {
  grid-column: span 2;
}

.trend-list {
  display: grid;
  gap: 10px;
}

.trend-list > div {
  display: grid;
  grid-template-columns: 110px 1fr;
  gap: 12px;
  align-items: center;
}

.site-table {
  border: 1px solid var(--app-border);
}

.pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.site-form :deep(.el-date-editor) {
  margin-right: 8px;
}

@media (max-width: 980px) {
  .analytics-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .page-head,
  .toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .content-tabs {
    width: 100%;
    min-width: 0;
  }

  .analytics-grid {
    grid-template-columns: 1fr;
  }

  .span-2 {
    grid-column: auto;
  }
}
</style>
