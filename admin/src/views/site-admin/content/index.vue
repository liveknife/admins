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
import { useI18n } from "@/i18n";
import { useUserStoreHook } from "@/store/modules/user";
import RePagination from "@/components/RePagination";

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

type UploadTarget = "image_url" | "cover_url" | "icon_url";

interface SiteForm {
  title: string;
  slug: string;
  content: string;
  markdown_content: string;
  link_url: string;
  is_active: boolean;
  sort_order: number;
  starts_at: string;
  ends_at: string;
  subtitle: string;
  image_url: string;
  summary: string;
  category: string;
  cover_url: string;
  tags: string;
  seo_title: string;
  seo_description: string;
  seo_keywords: string;
  status: string;
  is_featured: boolean;
  published_at: string;
  demo_url: string;
  repo_url: string;
  stack_tags: string;
  phase: string;
  event_type: string;
  happened_at: string;
  name: string;
  level: number;
  icon_url: string;
  description: string;
  visitor_name: string;
  email: string;
  reply: string;
  is_public: boolean;
}

const { t } = useI18n();
const userStore = useUserStoreHook();
const activeTab = ref<TabKey>("analytics");
const loading = ref(false);
const dialogVisible = ref(false);
const dialogMode = ref<"create" | "edit">("create");
const editingId = ref<number>();
const uploadTarget = ref<UploadTarget>("image_url");

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
const pagination = reactive({ page: 1, pageSize: 10, total: 0 });
const statusFilter = ref("");

const canWrite = computed(() => userStore.permissions.includes("site:write"));
const isTableTab = computed(() => activeTab.value !== "analytics");

const defaultForm = (): SiteForm => ({
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

const form = reactive<SiteForm>(defaultForm());

const currentRows = computed(() => {
  switch (activeTab.value) {
    case "announcements": return announcements.value;
    case "banners": return banners.value;
    case "resources": return resources.value;
    case "projects": return projects.value;
    case "timeline": return timeline.value;
    case "messages": return messages.value;
    case "tech": return techStacks.value;
    default: return [];
  }
});

const resetForm = () => Object.assign(form, defaultForm());

const fillForm = (row: Record<string, unknown>) => {
  resetForm();
  Object.keys(form).forEach(key => {
    if (row[key] !== undefined) (form as Record<string, unknown>)[key] = row[key] ?? "";
  });
};

const normalizePayload = () => {
  const payload = { ...form };
  (["starts_at", "ends_at", "published_at", "happened_at"] as const).forEach(key => {
    if (!payload[key]) (payload as Record<string, unknown>)[key] = null;
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
    switch (activeTab.value) {
      case "analytics":
        analytics.value = (await getSiteAnalytics()).analytics;
        break;
      case "announcements": {
        const r = await getSiteAnnouncements(params);
        announcements.value = r.announcements ?? [];
        totals.announcements = r.total ?? 0;
        break;
      }
      case "banners": {
        const r = await getSiteBanners(params);
        banners.value = r.banners ?? [];
        totals.banners = r.total ?? 0;
        break;
      }
      case "resources": {
        const r = await getSiteResources(params);
        resources.value = r.resources ?? [];
        totals.resources = r.total ?? 0;
        break;
      }
      case "projects": {
        const r = await getSiteProjects(params);
        projects.value = r.projects ?? [];
        totals.projects = r.total ?? 0;
        break;
      }
      case "timeline": {
        const r = await getSiteTimelineEvents(params);
        timeline.value = r.timeline ?? [];
        totals.timeline = r.total ?? 0;
        break;
      }
      case "messages": {
        const r = await getSiteMessages(params);
        messages.value = r.messages ?? [];
        totals.messages = r.total ?? 0;
        break;
      }
      default: {
        const r = await getSiteTechStacks(params);
        techStacks.value = r.tech_stacks ?? [];
        totals.tech = r.total ?? 0;
      }
    }
  } catch {
    message(t("site.loadFailed"), { type: "error" });
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

const openEdit = (row: Record<string, unknown>) => {
  fillForm(row);
  editingId.value = row.id as number;
  dialogMode.value = "edit";
  dialogVisible.value = true;
};

const saveCurrent = async () => {
  const payload = normalizePayload();
  try {
    switch (activeTab.value) {
      case "announcements": await saveSiteAnnouncement(payload, editingId.value); break;
      case "banners": await saveSiteBanner(payload, editingId.value); break;
      case "resources": await saveSiteResource(payload, editingId.value); break;
      case "projects": await saveSiteProject(payload, editingId.value); break;
      case "timeline": await saveSiteTimelineEvent(payload, editingId.value); break;
      case "messages": if (editingId.value) await saveSiteMessage(payload, editingId.value); break;
      default: await saveSiteTechStack(payload, editingId.value);
    }
    message(t("site.saveSuccess"), { type: "success" });
    dialogVisible.value = false;
    await loadData();
  } catch {
    message(t("site.saveFailed"), { type: "error" });
  }
};

const removeCurrent = async (row: { id: number }) => {
  try {
    switch (activeTab.value) {
      case "announcements": await deleteSiteAnnouncement(row.id); break;
      case "banners": await deleteSiteBanner(row.id); break;
      case "resources": await deleteSiteResource(row.id); break;
      case "projects": await deleteSiteProject(row.id); break;
      case "timeline": await deleteSiteTimelineEvent(row.id); break;
      case "messages": await deleteSiteMessage(row.id); break;
      default: await deleteSiteTechStack(row.id);
    }
    message(t("site.deleted"), { type: "success" });
    await loadData();
  } catch {
    /* user already sees error from API interceptor */
  }
};

const beforeUpload = async (file: File, target: UploadTarget) => {
  try {
    const res = await uploadSiteAsset(file);
    (form as unknown as Record<UploadTarget, string>)[target] = res.url;
    message(t("site.uploadDone"), { type: "success" });
  } catch {
    message(t("site.uploadFailed"), { type: "error" });
  }
  return false;
};

const formatTime = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";

const initTabFromQuery = () => {
  const tab = new URLSearchParams(window.location.search).get("tab") as TabKey;
  if (["analytics","announcements","banners","resources","projects","timeline","messages","tech"].includes(tab)) {
    activeTab.value = tab;
  }
};

onMounted(() => {
  initTabFromQuery();
  loadData();
});
</script>

<template>
  <div class="page-container">
    <!-- 头部 -->
    <div class="page-header">
      <div class="page-header-left">
        <h2 class="page-title">{{ t("site.title") }}</h2>
        <span class="page-badge">/api/v1/site</span>
      </div>
      <el-space v-if="isTableTab && activeTab !== 'messages' && canWrite">
        <el-button type="primary" @click="openCreate">
          + {{ t("common.add") }}
        </el-button>
      </el-space>
    </div>

    <!-- Tab + 工具栏 -->
    <div class="table-card toolbar-card">
      <div class="toolbar-inner">
        <el-tabs v-model="activeTab" class="content-tabs" @tab-change="handleTabChange">
          <el-tab-pane :label="t('site.tab.analytics')" name="analytics" />
          <el-tab-pane :label="t('site.tab.resources')" name="resources" />
          <el-tab-pane :label="t('site.tab.messages')" name="messages" />
          <el-tab-pane :label="t('site.tab.timeline')" name="timeline" />
          <el-tab-pane :label="t('site.tab.announcements')" name="announcements" />
          <el-tab-pane :label="t('site.tab.banners')" name="banners" />
          <el-tab-pane :label="t('site.tab.projects')" name="projects" />
          <el-tab-pane :label="t('site.tab.tech')" name="tech" />
        </el-tabs>
        <div class="toolbar-actions">
          <el-select
            v-if="isTableTab"
            v-model="statusFilter"
            clearable
            :placeholder="t('site.statusFilter')"
            style="width: 140px"
            @change="() => { pagination.page = 1; loadData(); }"
          >
            <template v-if="!['resources','projects','timeline','messages'].includes(activeTab)">
              <el-option :label="t('site.status.active')" value="active" />
              <el-option :label="t('site.status.inactive')" value="inactive" />
            </template>
            <template v-if="['resources','projects','timeline'].includes(activeTab)">
              <el-option :label="t('site.status.published')" value="published" />
              <el-option :label="t('site.status.draft')" value="draft" />
            </template>
            <template v-if="activeTab === 'messages'">
              <el-option :label="t('site.status.pending')" value="pending" />
              <el-option :label="t('site.status.approved')" value="approved" />
              <el-option :label="t('site.status.rejected')" value="rejected" />
            </template>
          </el-select>
          <el-button :loading="loading" @click="loadData">{{ t("common.refresh") }}</el-button>
        </div>
      </div>
    </div>

    <!-- Analytics 面板 -->
    <div v-if="activeTab === 'analytics'" v-loading="loading" class="analytics-grid">
      <div class="metric-card"><span>{{ t("site.analytics.visits") }}</span><strong>{{ analytics?.visit_count ?? 0 }}</strong></div>
      <div class="metric-card"><span>{{ t("site.analytics.today") }}</span><strong>{{ analytics?.today_visits ?? 0 }}</strong></div>
      <div class="metric-card"><span>{{ t("site.analytics.articles") }}</span><strong>{{ analytics?.article_count ?? 0 }}</strong></div>
      <div class="metric-card"><span>{{ t("site.analytics.pendingMsg") }}</span><strong>{{ analytics?.pending_messages ?? 0 }}</strong></div>
      <div class="panel-card span-2">
        <h3>{{ t("site.analytics.trend7d") }}</h3>
        <div class="trend-list">
          <div v-for="item in analytics?.visits_by_day" :key="item.date">
            <span>{{ item.date }}</span>
            <el-progress :percentage="Math.min(item.visits * 10, 100)" :format="() => `${item.visits}`" />
          </div>
        </div>
      </div>
      <div class="panel-card">
        <h3>{{ t("site.analytics.topPages") }}</h3>
        <p v-for="item in analytics?.top_pages" :key="item.path">
          <span>{{ item.path }}</span><strong>{{ item.visits }}</strong>
        </p>
      </div>
      <div class="panel-card">
        <h3>{{ t("site.analytics.devices") }}</h3>
        <p v-for="item in analytics?.device_stats" :key="item.device">
          <span>{{ item.device }}</span><strong>{{ item.visits }}</strong>
        </p>
      </div>
      <div class="panel-card span-2">
        <h3>{{ t("site.analytics.topArticles") }}</h3>
        <p v-for="item in analytics?.top_articles" :key="item.id">
          <span>{{ item.title }}</span><strong>{{ item.view_count }}</strong>
        </p>
      </div>
    </div>

    <!-- 数据表格 -->
    <template v-else>
      <div class="table-card">
        <el-table v-loading="loading" :data="currentRows" stripe class="admin-table" row-key="id">
          <el-table-column v-if="activeTab !== 'messages'" prop="sort_order" :label="t('site.cols.sortOrder')" width="80" />

          <el-table-column v-if="activeTab === 'tech'" prop="name" :label="t('site.cols.name')" min-width="140" />
          <el-table-column v-else-if="activeTab === 'projects'" prop="name" :label="t('site.cols.projectName')" min-width="180" />
          <el-table-column v-else-if="activeTab === 'messages'" prop="visitor_name" :label="t('site.cols.visitor')" width="140" />
          <el-table-column v-else prop="title" :label="t('site.cols.title')" min-width="180" />

          <el-table-column v-if="activeTab === 'resources'" prop="slug" :label="t('site.cols.title')" min-width="160" />
          <el-table-column v-if="activeTab === 'banners'" prop="subtitle" :label="t('site.cols.subtitle')" min-width="220" />
          <el-table-column v-if="activeTab === 'timeline'" prop="phase" :label="t('site.cols.phase')" width="120" />
          <el-table-column v-if="activeTab === 'timeline'" prop="event_type" :label="t('site.cols.eventType')" width="120" />
          <el-table-column v-if="['resources','projects','timeline'].includes(activeTab)" prop="summary" :label="t('site.cols.summary')" min-width="260" />
          <el-table-column v-if="activeTab === 'messages'" prop="content" :label="t('site.cols.messageContent')" min-width="260" show-overflow-tooltip />
          <el-table-column v-if="activeTab === 'messages'" prop="reply" :label="t('site.cols.reply')" min-width="220" show-overflow-tooltip />
          <el-table-column v-if="activeTab === 'projects'" prop="stack_tags" :label="t('site.cols.stackTags')" min-width="220" />
          <el-table-column v-if="activeTab === 'timeline'" prop="tags" :label="t('site.cols.tags')" min-width="180" />
          <el-table-column v-if="['tech','resources'].includes(activeTab)" prop="category" :label="t('site.cols.category')" width="120" />
          <el-table-column v-if="activeTab === 'tech'" prop="level" :label="t('site.cols.proficiency')" width="120">
            <template #default="{ row }">
              <el-progress :percentage="row.level" :stroke-width="8" />
            </template>
          </el-table-column>
          <el-table-column v-if="activeTab === 'resources'" prop="view_count" :label="t('site.cols.views')" width="90" />
          <el-table-column v-if="activeTab === 'timeline'" :label="t('site.cols.happenedAt')" width="170">
            <template #default="{ row }">{{ formatTime(row.happened_at) }}</template>
          </el-table-column>

          <el-table-column v-if="['resources','projects','timeline','messages'].includes(activeTab)" prop="status" :label="t('site.cols.status')" width="100">
            <template #default="{ row }">
              <el-tag :type="['published','approved'].includes(row.status) ? 'success' : row.status === 'rejected' ? 'danger' : 'info'" size="small">
                {{ row.status === 'published' ? t('site.status.published') : row.status === 'draft' ? t('site.status.draft') : row.status === 'approved' ? t('site.status.approved') : row.status === 'pending' ? t('site.status.pending') : row.status }}
              </el-tag>
            </template>
          </el-table-column>

          <el-table-column v-if="!['resources','projects','timeline','messages'].includes(activeTab)" prop="is_active" :label="t('site.cols.enabled')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.is_active ? 'success' : 'info'" size="small">{{ row.is_active ? t('site.status.active') : t('site.status.inactive') }}</el-tag>
            </template>
          </el-table-column>

          <el-table-column :label="t('site.cols.updatedAt')" width="170">
            <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
          </el-table-column>

          <el-table-column v-if="canWrite" :label="t('common.operation')" fixed="right" width="150">
            <template #default="{ row }">
              <el-button link type="primary" @click="openEdit(row)">
                {{ activeTab === "messages" ? t("site.actionReply") : t("common.edit") }}
              </el-button>
              <el-popconfirm :title="t('site.deleteConfirm')" @confirm="removeCurrent(row)">
                <template #reference>
                  <el-button link type="danger">{{ t("common.delete") }}</el-button>
                </template>
              </el-popconfirm>
            </template>
          </el-table-column>
        </el-table>

        <RePagination
          v-model:page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="totals[activeTab]"
          :disabled="loading"
          @change="loadData"
        />
      </div>
    </template>

    <!-- ═══ 新增 / 编辑弹窗 ═══ -->
    <el-dialog
      v-model="dialogVisible"
      :title="t(activeTab === 'messages' ? 'site.actionReply' : dialogMode === 'create' ? 'common.add' : 'common.edit')"
      width="820px"
      destroy-on-close
    >
      <el-form :model="form" label-width="104px" class="site-form">

        <!-- ── 公告 ── -->
        <template v-if="activeTab === 'announcements'">
          <el-form-item :label="t('common.title')"><el-input v-model="form.title" /></el-form-item>
          <el-form-item :label="t('site.form.markdown')"><el-input v-model="form.content" type="textarea" :rows="4" /></el-form-item>
          <el-form-item :label="t('site.form.linkUrl')"><el-input v-model="form.link_url" /></el-form-item>
          <el-form-item :label="t('site.form.startTime') + ' / ' + t('site.form.endTime')">
            <el-date-picker v-model="form.starts_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" :placeholder="t('site.form.startTime')" />
            <el-date-picker v-model="form.ends_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" :placeholder="t('site.form.endTime')" />
          </el-form-item>
        </template>

        <!-- ── 轮播图 ── -->
        <template v-if="activeTab === 'banners'">
          <el-form-item :label="t('common.title')"><el-input v-model="form.title" /></el-form-item>
          <el-form-item :label="t('site.cols.subtitle')"><el-input v-model="form.subtitle" /></el-form-item>
          <el-form-item :label="t('site.form.image')">
            <el-input v-model="form.image_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="(f: File) => { uploadTarget = 'image_url'; return beforeUpload(f, 'image_url'); }">
                  <el-button>{{ t("site.upload") }}</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item :label="t('site.form.linkUrl')"><el-input v-model="form.link_url" /></el-form-item>
        </template>

        <!-- ── 博客文章 ── -->
        <template v-if="activeTab === 'resources'">
          <el-form-item :label="t('common.title')"><el-input v-model="form.title" /></el-form-item>
          <el-form-item :label="t('site.form.slug')"><el-input v-model="form.slug" placeholder="react-hooks-notes" /></el-form-item>
          <el-form-item :label="t('site.form.summary')"><el-input v-model="form.summary" type="textarea" :rows="3" /></el-form-item>
          <el-form-item :label="t('site.form.markdown')"><el-input v-model="form.markdown_content" type="textarea" :rows="9" /></el-form-item>
          <el-form-item :label="t('site.form.renderedContent')"><el-input v-model="form.content" type="textarea" :rows="5" /></el-form-item>
          <el-form-item :label="t('site.cols.category')">
            <el-select v-model="form.category">
              <el-option :label="t('site.category.frontend')" value="frontend" />
              <el-option :label="t('site.category.backend')" value="backend" />
              <el-option :label="t('site.category.database')" value="database" />
              <el-option :label="t('site.category.learning')" value="learning" />
              <el-option :label="t('site.category.project')" value="project" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('site.form.cover')">
            <el-input v-model="form.cover_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="(f: File) => { uploadTarget = 'cover_url'; return beforeUpload(f, 'cover_url'); }">
                  <el-button>{{ t("site.upload") }}</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item :label="t('site.form.tags')"><el-input v-model="form.tags" :placeholder="t('site.form.tagsPlaceholder')" /></el-form-item>
          <el-form-item :label="t('site.form.linkUrl')"><el-input v-model="form.link_url" /></el-form-item>
          <el-form-item :label="t('site.form.seoTitle')"><el-input v-model="form.seo_title" /></el-form-item>
          <el-form-item :label="t('site.form.seoDesc')"><el-input v-model="form.seo_description" type="textarea" :rows="2" /></el-form-item>
          <el-form-item :label="t('site.form.seoKeywords')"><el-input v-model="form.seo_keywords" placeholder="React,Hooks,Vite" /></el-form-item>
          <el-form-item :label="t('site.form.publishStatus')">
            <el-segmented v-model="form.status" :options="[{ label: t('site.status.published'), value: 'published' }, { label: t('site.status.draft'), value: 'draft' }]" />
          </el-form-item>
          <el-form-item :label="t('site.form.featured')"><el-switch v-model="form.is_featured" /></el-form-item>
        </template>

        <!-- ── 项目 Demo ── -->
        <template v-if="activeTab === 'projects'">
          <el-form-item :label="t('site.cols.projectName')"><el-input v-model="form.name" /></el-form-item>
          <el-form-item :label="t('site.form.summary')"><el-input v-model="form.summary" type="textarea" :rows="3" /></el-form-item>
          <el-form-item :label="t('site.form.description')"><el-input v-model="form.description" type="textarea" :rows="5" /></el-form-item>
          <el-form-item :label="t('site.form.cover')">
            <el-input v-model="form.cover_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="(f: File) => { uploadTarget = 'cover_url'; return beforeUpload(f, 'cover_url'); }">
                  <el-button>{{ t("site.upload") }}</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item :label="t('site.form.demoUrl')"><el-input v-model="form.demo_url" /></el-form-item>
          <el-form-item :label="t('site.form.repoUrl')"><el-input v-model="form.repo_url" /></el-form-item>
          <el-form-item :label="t('site.form.tags')"><el-input v-model="form.stack_tags" placeholder="React,Three.js,Go Gin" /></el-form-item>
          <el-form-item :label="t('site.form.publishStatus')">
            <el-segmented v-model="form.status" :options="[{ label: t('site.status.published'), value: 'published' }, { label: t('site.status.draft'), value: 'draft' }]" />
          </el-form-item>
          <el-form-item :label="t('site.form.featured')"><el-switch v-model="form.is_featured" /></el-form-item>
        </template>

        <!-- ── 时间轴 ── -->
        <template v-if="activeTab === 'timeline'">
          <el-form-item :label="t('common.title')"><el-input v-model="form.title" /></el-form-item>
          <el-form-item :label="t('site.form.summary')"><el-input v-model="form.summary" type="textarea" :rows="3" /></el-form-item>
          <el-form-item :label="t('site.cols.content')"><el-input v-model="form.content" type="textarea" :rows="5" /></el-form-item>
          <el-form-item :label="t('site.cols.phase')"><el-input v-model="form.phase" placeholder="Foundation / Growth / Release" /></el-form-item>
          <el-form-item :label="t('site.cols.eventType')">
            <el-select v-model="form.event_type">
              <el-option :label="t('site.eventType.learning')" value="learning" />
              <el-option :label="t('site.eventType.project')" value="project" />
              <el-option :label="t('site.eventType.release')" value="release" />
              <el-option :label="t('site.eventType.pitfall')" value="pitfall" />
              <el-option :label="t('site.eventType.review')" value="review" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('site.form.tags')"><el-input v-model="form.tags" placeholder="React,Go,RBAC" /></el-form-item>
          <el-form-item :label="t('site.form.linkUrl')"><el-input v-model="form.link_url" /></el-form-item>
          <el-form-item :label="t('site.form.selectTime')">
            <el-date-picker v-model="form.happened_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" :placeholder="t('site.form.selectTime')" />
          </el-form-item>
          <el-form-item :label="t('site.form.publishStatus')">
            <el-segmented v-model="form.status" :options="[{ label: t('site.status.published'), value: 'published' }, { label: t('site.status.draft'), value: 'draft' }]" />
          </el-form-item>
          <el-form-item :label="t('site.form.featured')"><el-switch v-model="form.is_featured" /></el-form-item>
        </template>

        <!-- ── 留言审核 ── -->
        <template v-if="activeTab === 'messages'">
          <el-form-item :label="t('site.cols.visitor')"><el-input v-model="form.visitor_name" /></el-form-item>
          <el-form-item :label="t('site.form.email')"><el-input v-model="form.email" /></el-form-item>
          <el-form-item :label="t('site.cols.messageContent')"><el-input v-model="form.content" type="textarea" :rows="4" /></el-form-item>
          <el-form-item :label="t('site.cols.reply')"><el-input v-model="form.reply" type="textarea" :rows="4" /></el-form-item>
          <el-form-item :label="t('site.form.reviewStatus')">
            <el-segmented v-model="form.status" :options="[{ label: t('site.status.pending'), value: 'pending' }, { label: t('site.status.approved'), value: 'approved' }, { label: t('site.status.rejected'), value: 'rejected' }]" />
          </el-form-item>
          <el-form-item :label="t('site.form.publicDisplay')"><el-switch v-model="form.is_public" /></el-form-item>
        </template>

        <!-- ── 技术栈 ── -->
        <template v-if="activeTab === 'tech'">
          <el-form-item :label="t('site.cols.name')"><el-input v-model="form.name" /></el-form-item>
          <el-form-item :label="t('site.cols.category')"><el-input v-model="form.category" /></el-form-item>
          <el-form-item :label="t('site.cols.proficiency')"><el-slider v-model="form.level" :min="0" :max="100" /></el-form-item>
          <el-form-item :label="t('site.form.icon')">
            <el-input v-model="form.icon_url">
              <template #append>
                <el-upload :show-file-list="false" :before-upload="(f: File) => { uploadTarget = 'icon_url'; return beforeUpload(f, 'icon_url'); }">
                  <el-button>{{ t("site.upload") }}</el-button>
                </el-upload>
              </template>
            </el-input>
          </el-form-item>
          <el-form-item :label="t('site.form.description')"><el-input v-model="form.description" type="textarea" :rows="3" /></el-form-item>
        </template>

        <!-- ── 公共字段 ── -->
        <el-form-item v-if="!['resources','projects','timeline','messages'].includes(activeTab)" :label="t('site.cols.enabled')">
          <el-switch v-model="form.is_active" />
        </el-form-item>
        <el-form-item v-if="activeTab !== 'messages'" :label="t('site.cols.sortOrder')">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :disabled="!canWrite" @click="saveCurrent">{{ t("common.save") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.page-container {
  padding: 24px 28px;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.page-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.page-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--app-text);
  margin: 0;
}

.page-badge {
  display: inline-block;
  padding: 2px 8px;
  background: var(--app-bg-soft);
  color: var(--app-violet);
  font-size: 11.5px;
  font-weight: 500;
  border-radius: 4px;
  border: 1px solid #e0e7ff;
  font-family: "Menlo", "Monaco", monospace;
}

.table-card {
  background: var(--app-surface);
  border-radius: 8px;
  border: 1px solid var(--app-border-soft);
  overflow: hidden;
  box-shadow: 0 10px 28px rgb(33 49 77 / 8%);
}

.toolbar-card {
  margin-bottom: 16px;
}

.toolbar-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 12px;
  flex-wrap: wrap;
}

.content-tabs {
  min-width: 520px;

  :deep(.el-tabs__header) {
    margin-bottom: 0;
  }

  :deep(.el-tabs__nav-wrap::after) {
    display: none;
  }
}

.toolbar-actions {
  display: flex;
  gap: 10px;
}

/* ── Analytics 面板 ── */
.analytics-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card,
.panel-card {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
  padding: 18px;
  box-shadow: 0 6px 20px rgb(33 49 77 / 5%);
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
  font-size: 15px;
  font-weight: 700;
  color: var(--app-text);
}

.panel-card p {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin: 10px 0;
  color: var(--app-text-secondary);
  font-size: 13px;
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

/* ── 表格 ── */
.admin-table {
  --el-table-header-bg-color: var(--app-surface-muted);
  --el-table-header-text-color: var(--app-text-secondary);
  --el-table-row-hover-bg-color: var(--app-surface-soft);

  :deep(.el-table__header th) {
    height: 46px;
    font-weight: 700;
    background: var(--app-surface-muted) !important;
  }

  :deep(.el-table__row) {
    height: 56px;
  }

  :deep(.el-table__cell) {
    border-color: var(--app-border-soft);
  }
}

/* ── 表单 ── */
.site-form :deep(.el-date-editor) {
  margin-right: 8px;
}

/* ── 响应式 ── */
@media (max-width: 1100px) {
  .toolbar-inner {
    flex-direction: column;
    align-items: stretch;
  }

  .content-tabs {
    min-width: 0;
    width: 100%;
  }

  .toolbar-actions {
    justify-content: flex-end;
  }
}

@media (max-width: 980px) {
  .analytics-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .page-container {
    padding: 16px;
  }

  .analytics-grid {
    grid-template-columns: 1fr;
  }

  .span-2 {
    grid-column: auto;
  }
}
</style>
