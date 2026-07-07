import { http } from "@/utils/http";
import { encryptPassword } from "@/utils/passwordCrypto";
import type { GoUser } from "./user";

export type GoRole = {
  id: number;
  name: string;
  description: string;
  permissions: string[];
  created_at: string;
};

export type GoPermission = {
  id: number;
  code: string;
  description: string;
  created_at: string;
};

export type DashboardMetric = {
  date: string;
  users: number;
  messages: number;
  logs: number;
};

export type OperationLog = {
  id: number;
  user_id: number;
  username: string;
  action: string;
  resource: string;
  detail: string;
  ip: string;
  user_agent: string;
  created_at: string;
};

export type Notification = {
  id: number;
  user_id?: number;
  title: string;
  content: string;
  type: string;
  is_read: boolean;
  created_at: string;
  read_at?: string;
};

export type AdminAnnouncement = {
  id: number;
  title: string;
  content: string;
  type: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type DashboardSummary = {
  user_count: number;
  active_user_count: number;
  role_count: number;
  permission_count: number;
  message_count: number;
  unread_notification: number;
  recent_logs: OperationLog[];
  recent_notifications: Notification[];
  metric_trend: DashboardMetric[];
  // 系统资源监控
  system_resources?: {
    cpu_usage: number;
    memory_usage: number;
    disk_usage: number;
    cpu_trend: number[];
    memory_trend: number[];
  };
  // 访问来源分布
  source_stats?: { name: string; value: number }[];
  // 消息类型统计
  message_type_stats?: { name: string; value: number }[];
};

export type PermissionTreeNode = {
  id: string;
  label: string;
  code?: string;
  type: string;
  description?: string;
  children?: PermissionTreeNode[];
};

export type RolePermissionPreview = {
  role: GoRole;
  menus: PermissionTreeNode[];
  buttons: PermissionTreeNode[];
  permissions: string[];
};

export type KnowledgeSource = {
  chunk_id?: number;
  citation_id?: number;
  source_type: string;
  source_id: number;
  visibility?: string;
  title: string;
  summary: string;
  score: number;
  vector_score?: number;
  bm25_score?: number;
  keyword_score?: number;
  source_weight?: number;
  rerank_score?: number;
  threshold?: number;
  url?: string;
  snippet?: string;
  highlighted_text?: string;
};

export type AIAssistantResult = {
  question: string;
  answer: string;
  insights: string[];
  rows: Array<Record<string, any>>;
  metrics: Record<string, number>;
  sources?: KnowledgeSource[];
};

export type RAGIndexStats = {
  total_chunks: number;
  by_source: Record<string, number>;
  by_visibility: Record<string, number>;
  top_k: number;
  min_score: number;
  rerank_top_n: number;
  source_weights: Record<string, number>;
  chat_enabled: boolean;
  streaming_enabled: boolean;
  vector_backend: string;
  pgvector_available: boolean;
  updated_at?: string;
  latest_job?: RAGIndexJob;
  query_count: number;
  hit_count: number;
  average_latency_ms: number;
  average_source_count: number;
  feedback_count: number;
  positive_feedback: number;
  negative_feedback: number;
};

export type RAGIndexJob = {
  id: number;
  job_type: string;
  status: string;
  retry_count: number;
  max_retries: number;
  error_message: string;
  started_at?: string;
  finished_at?: string;
  created_at: string;
  updated_at: string;
};

export type RAGQueryLog = {
  id: number;
  question: string;
  answer: string;
  matched: boolean;
  source_count: number;
  top_score: number;
  latency_ms: number;
  used_chat_model: boolean;
  source_json: string;
  created_at: string;
};

export type RAGFeedback = {
  id: number;
  query_log_id: number;
  question: string;
  rating: string;
  comment: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
};

export type KnowledgeChunkPreview = {
  id: number;
  source_type: string;
  source_id: number;
  visibility: string;
  title: string;
  summary: string;
  content: string;
  metadata?: Record<string, any>;
  token_count: number;
  status: string;
  created_at: string;
  updated_at: string;
};

export type RAGEvalCase = {
  id: string;
  question: string;
  expected_sources?: string[];
  expected_terms?: string[];
};

export type RAGEvalCaseResult = {
  case: RAGEvalCase;
  matched: boolean;
  recall_hit: boolean;
  answer_quality: number;
  top_score: number;
  latency_ms: number;
  sources: KnowledgeSource[];
  answer: string;
};

export type RAGEvalRun = {
  total: number;
  matched: number;
  recall_hits: number;
  average_top_score: number;
  average_quality: number;
  average_latency_ms: number;
  results: RAGEvalCaseResult[];
  created_at: string;
};

export type UploadedDocument = {
  id: number;
  original_name: string;
  file_name: string;
  file_path: string;
  mime_type: string;
  file_size: number;
  visibility: string;
  text_content: string;
  chunk_count: number;
  status: string;
  error_message: string;
  created_at: string;
  updated_at: string;
};

export type AIModelConfig = {
  id: number;
  name: string;
  provider: string;
  api_format: string;
  base_url: string;
  chat_model: string;
  embedding_model: string;
  api_key?: string;
  has_api_key: boolean;
  masked_api_key?: string;
  temperature: number;
  max_tokens: number;
  timeout_seconds: number;
  extra_json: string;
  is_default: boolean;
  enabled: boolean;
  last_test_status: string;
  last_test_message: string;
  last_test_at?: string;
  created_at: string;
  updated_at: string;
};

export type SystemHealth = {
  status: string;
  cpu: { value: number; unit: string; label: string };
  memory: { value: number; unit: string; label: string };
  database: {
    status: string;
    open_connection: number;
    in_use: number;
    idle: number;
    wait_count: number;
    ping_ms: number;
  };
  websocket: {
    online_users: number;
    connections: number;
  };
  api: {
    total_requests: number;
    average_ms: number;
    slow_requests: number;
    status_codes: Record<string, number>;
    top_paths: Array<{
      path: string;
      method: string;
      count: number;
      average_ms: number;
    }>;
  };
  alerts: string[];
  checked_at: string;
};

export type DatabaseCatalog = {
  current_database: string;
  databases: string[];
  engines: string[];
};

export type DatabaseTable = {
  name: string;
  engine: string;
  collation: string;
  rows: number;
  index_size: string;
  comment: string;
  created_at?: string;
};

export type DatabaseColumn = {
  name: string;
  type: string;
  not_null: boolean;
  default: string;
  comment: string;
  primary_key: boolean;
};

export type SiteAnnouncement = {
  id: number;
  title: string;
  content: string;
  link_url: string;
  is_active: boolean;
  sort_order: number;
  starts_at?: string;
  ends_at?: string;
  created_at: string;
  updated_at: string;
};

export type SiteBanner = {
  id: number;
  title: string;
  subtitle: string;
  image_url: string;
  link_url: string;
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
};

export type SiteResource = {
  id: number;
  title: string;
  slug: string;
  summary: string;
  content: string;
  markdown_content: string;
  category: string;
  cover_url: string;
  link_url: string;
  tags: string;
  seo_title: string;
  seo_description: string;
  seo_keywords: string;
  status: string;
  is_featured: boolean;
  view_count: number;
  sort_order: number;
  published_at?: string;
  created_at: string;
  updated_at: string;
};

export type SiteTechStack = {
  id: number;
  name: string;
  category: string;
  level: number;
  icon_url: string;
  description: string;
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
};

export type SiteProject = {
  id: number;
  name: string;
  summary: string;
  description: string;
  cover_url: string;
  demo_url: string;
  repo_url: string;
  stack_tags: string;
  status: string;
  is_featured: boolean;
  sort_order: number;
  published_at?: string;
  created_at: string;
  updated_at: string;
};

export type SiteTimelineEvent = {
  id: number;
  title: string;
  summary: string;
  content: string;
  phase: string;
  event_type: string;
  tags: string;
  link_url: string;
  status: string;
  is_featured: boolean;
  sort_order: number;
  happened_at?: string;
  published_at?: string;
  created_at: string;
  updated_at: string;
};

export type SiteMessage = {
  id: number;
  visitor_name: string;
  email: string;
  content: string;
  reply: string;
  status: string;
  is_public: boolean;
  ip_address: string;
  user_agent: string;
  created_at: string;
  updated_at: string;
};

export type SiteAnalytics = {
  visit_count: number;
  today_visits: number;
  article_count: number;
  message_count: number;
  pending_messages: number;
  visits_by_day: Array<{ date: string; visits: number }>;
  top_pages: Array<{ path: string; visits: number }>;
  device_stats: Array<{ device: string; visits: number }>;
  top_articles: SiteResource[];
};

export type RolePayload = {
  name: string;
  description?: string;
  permissions: string[];
};

export type CreateUserPayload = {
  username: string;
  email: string;
  phone?: string;
  password_encrypted: string;
  roles?: string[];
};

export type UpdateUserPayload = {
  username: string;
  email: string;
  phone?: string;
};

export type PageParams = {
  page?: number;
  page_size?: number;
};

export type PagedResult<T extends string, R> = Record<T, R[]> & {
  total: number;
  page: number;
  page_size: number;
};

export const getAdminUsers = (params?: PageParams) => {
  return http.request<PagedResult<"users", GoUser>>(
    "get",
    "/api/v1/admin/users",
    { params }
  );
};

// 管理员创建用户
export const createAdminUser = async (data: CreateUserPayload) => {
  return http.request<{ user: GoUser }>("post", "/api/v1/admin/users", {
    data
  });
};

// 管理员编辑用户信息
export const updateAdminUser = (id: number, data: UpdateUserPayload) => {
  return http.request<{ user: GoUser }>(
    "put",
    `/api/v1/admin/users/${id}`,
    { data }
  );
};

export const setAdminUserRoles = (id: number, roles: string[]) => {
  return http.request<{ user: GoUser }>(
    "put",
    `/api/v1/admin/users/${id}/roles`,
    { data: { roles } }
  );
};

export const getAdminUserPassword = (id: number) => {
  return http.request<{ password: string }>(
    "get",
    `/api/v1/admin/users/${id}/password`
  );
};

export const resetAdminUserPassword = async (id: number, password: string) => {
  const passwordEncrypted = await encryptPassword(password);
  return http.request<{ message: string }>(
    "put",
    `/api/v1/admin/users/${id}/password`,
    { data: { password_encrypted: passwordEncrypted } }
  );
};

export const deactivateAdminUser = (id: number) => {
  return http.request<{ message: string }>("delete", `/api/v1/admin/users/${id}`);
};

// 管理员永久删除用户（硬删除，不可恢复）
export const deleteAdminUser = (id: number) => {
  return http.request<{ message: string }>("delete", `/api/v1/admin/users/${id}/permanent`);
};

// 恢复已禁用用户并重置密码为默认密码
export const reactivateAdminUser = (id: number) => {
  return http.request<{ message: string; user: GoUser }>(
    "put",
    `/api/v1/admin/users/${id}/reactivate`
  );
};

export const getAdminRoles = (params?: PageParams) => {
  return http.request<PagedResult<"roles", GoRole>>(
    "get",
    "/api/v1/admin/roles",
    { params }
  );
};

export const createAdminRole = (data: RolePayload) => {
  return http.request<{ role: GoRole }>("post", "/api/v1/admin/roles", {
    data
  });
};

export const updateAdminRole = (id: number, data: RolePayload) => {
  return http.request<{ role: GoRole }>("put", `/api/v1/admin/roles/${id}`, {
    data
  });
};

export const deleteAdminRole = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/roles/${id}`);
};

export const getAdminPermissions = () => {
  return http.request<{ permissions: GoPermission[] }>(
    "get",
    "/api/v1/admin/permissions"
  );
};

export const getPermissionTree = () => {
  return http.request<{ tree: PermissionTreeNode[] }>(
    "get",
    "/api/v1/admin/permissions/tree"
  );
};

export const getRolePreview = (id: number) => {
  return http.request<{ preview: RolePermissionPreview }>(
    "get",
    `/api/v1/admin/roles/${id}/preview`
  );
};

export const getAdminDashboard = () => {
  return http.request<{ dashboard: DashboardSummary }>(
    "get",
    "/api/v1/admin/dashboard"
  );
};

export const getOperationLogs = (params?: PageParams) => {
  return http.request<PagedResult<"logs", OperationLog>>(
    "get",
    "/api/v1/admin/operation-logs",
    { params }
  );
};

export const getNotifications = (params?: PageParams & { read_status?: string }) => {
  return http.request<PagedResult<"notifications", Notification>>(
    "get",
    "/api/v1/admin/notifications",
    { params }
  );
};

export const getUnreadNotificationCount = () => {
  return http.request<{ count: number }>(
    "get",
    "/api/v1/admin/notifications/unread-count"
  );
};

export const markNotificationRead = (id: number) => {
  return http.request<{ message: string }>(
    "put",
    `/api/v1/admin/notifications/${id}/read`
  );
};

export const markAllNotificationsRead = () => {
  return http.request<{ message: string }>(
    "put",
    "/api/v1/admin/notifications/read-all"
  );
};

export const createNotification = (data: {
  title: string;
  content: string;
  type: string;
}) => {
  return http.request<{ notification: Notification }>(
    "post",
    "/api/v1/admin/notifications",
    { data }
  );
};

export const deleteNotification = (id: number) => {
  return http.request<{ message: string }>(
    "delete",
    `/api/v1/admin/notifications/${id}`
  );
};

// ── 后台公告 API ──
export const getAnnouncements = (params?: PageParams) => {
  return http.request<PagedResult<"announcements", AdminAnnouncement>>(
    "get",
    "/api/v1/admin/announcements",
    { params }
  );
};

// 公开公告列表（所有已登录用户可查看，无需特殊权限）
export const getPublicAnnouncements = () => {
  return http.request<{ announcements: AdminAnnouncement[] }>(
    "get",
    "/api/v1/admin/announcements/public"
  );
};

export const createAnnouncement = (data: {
  title: string;
  content: string;
  type: string;
  is_active: boolean;
}) => {
  return http.request<{ announcement: AdminAnnouncement }>(
    "post",
    "/api/v1/admin/announcements",
    { data }
  );
};

export const updateAnnouncement = (
  id: number,
  data: {
    title: string;
    content: string;
    type: string;
    is_active: boolean;
  }
) => {
  return http.request<{ announcement: AdminAnnouncement }>(
    "put",
    `/api/v1/admin/announcements/${id}`,
    { data }
  );
};

export const deleteAnnouncement = (id: number) => {
  return http.request<{ message: string }>(
    "delete",
    `/api/v1/admin/announcements/${id}`
  );
};

export const askAIAssistant = (question: string) => {
  return http.request<{ result: AIAssistantResult }>(
    "post",
    "/api/v1/admin/ai/ask",
    { data: { question } }
  );
};

export const getAIModelConfigs = () => {
  return http.request<{ configs: AIModelConfig[] }>(
    "get",
    "/api/v1/admin/ai/model-configs"
  );
};

export const saveAIModelConfig = (
  data: Partial<AIModelConfig>,
  id?: number
) => {
  return http.request<{ config: AIModelConfig }>(
    id ? "put" : "post",
    id
      ? `/api/v1/admin/ai/model-configs/${id}`
      : "/api/v1/admin/ai/model-configs",
    { data }
  );
};

export const setDefaultAIModelConfig = (id: number) => {
  return http.request<{ config: AIModelConfig }>(
    "post",
    `/api/v1/admin/ai/model-configs/${id}/default`
  );
};

export const testAIModelConfig = (id: number) => {
  return http.request<{ config: AIModelConfig }>(
    "post",
    `/api/v1/admin/ai/model-configs/${id}/test`
  );
};

export const deleteAIModelConfig = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/ai/model-configs/${id}`);
};

export const getRAGIndexStats = () => {
  return http.request<{ stats: RAGIndexStats }>(
    "get",
    "/api/v1/admin/ai/rag/stats"
  );
};

export const rebuildRAGIndex = () => {
  return http.request<{ job: RAGIndexJob }>(
    "post",
    "/api/v1/admin/ai/rag/reindex"
  );
};

export const getRAGIndexJobs = (limit = 20) => {
  return http.request<{ jobs: RAGIndexJob[] }>(
    "get",
    "/api/v1/admin/ai/rag/jobs",
    { params: { limit } }
  );
};

export const retryRAGIndexJob = (id: number) => {
  return http.request<{ job: RAGIndexJob }>(
    "post",
    `/api/v1/admin/ai/rag/jobs/${id}/retry`
  );
};

export const getRAGQueryLogs = (limit = 30) => {
  return http.request<{ logs: RAGQueryLog[] }>(
    "get",
    "/api/v1/admin/ai/rag/query-logs",
    { params: { limit } }
  );
};

export const getRAGFeedback = (params?: { limit?: number; rating?: string }) => {
  return http.request<{ feedback: RAGFeedback[] }>(
    "get",
    "/api/v1/admin/ai/rag/feedback",
    { params }
  );
};

export const searchRAGDiagnostics = (data: {
  question: string;
  include_internal?: boolean;
  top_k?: number;
}) => {
  return http.request<{ sources: KnowledgeSource[] }>(
    "post",
    "/api/v1/admin/ai/rag/diagnostics",
    { data }
  );
};

export const runRAGEval = (includeInternal = true) => {
  return http.request<{ run: RAGEvalRun }>(
    "post",
    "/api/v1/admin/ai/rag/evals/run",
    { params: { include_internal: includeInternal } }
  );
};

export const getRAGChunks = (params?: {
  source_type?: string;
  source_id?: number;
  limit?: number;
}) => {
  return http.request<{ chunks: KnowledgeChunkPreview[] }>(
    "get",
    "/api/v1/admin/ai/rag/chunks",
    { params }
  );
};

export const getUploadedDocuments = (params?: PageParams) => {
  return http.request<PagedResult<"documents", UploadedDocument>>(
    "get",
    "/api/v1/admin/documents",
    { params }
  );
};

export const uploadDocument = (file: File, visibility = "internal") => {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("visibility", visibility);
  return http.request<{ document: UploadedDocument }>(
    "post",
    "/api/v1/admin/documents/upload",
    {
      data: formData,
      headers: {
        "Content-Type": "multipart/form-data"
      }
    }
  );
};

export const previewDocument = (id: number) => {
  return http.request<{ document: UploadedDocument }>(
    "get",
    `/api/v1/admin/documents/${id}/preview`
  );
};

export const getDocumentChunks = (id: number, limit = 100) => {
  return http.request<{ chunks: KnowledgeChunkPreview[] }>(
    "get",
    `/api/v1/admin/documents/${id}/chunks`,
    { params: { limit } }
  );
};

export const updateDocumentVisibility = (id: number, visibility: string) => {
  return http.request<{ document: UploadedDocument }>(
    "put",
    `/api/v1/admin/documents/${id}/visibility`,
    { data: { visibility } }
  );
};

export const rebuildDocument = (id: number) => {
  return http.request<{ document: UploadedDocument }>(
    "post",
    `/api/v1/admin/documents/${id}/rebuild`
  );
};

export const deleteDocument = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/documents/${id}`);
};

export const getSystemHealth = () => {
  return http.request<{ health: SystemHealth }>(
    "get",
    "/api/v1/admin/health"
  );
};

export const getDatabaseCatalog = () => {
  return http.request<{ catalog: DatabaseCatalog }>(
    "get",
    "/api/v1/admin/database/catalog"
  );
};

export const getDatabaseTables = (params?: {
  database?: string;
  table?: string;
  engine?: string;
  comment?: string;
}) => {
  return http.request<{ tables: DatabaseTable[] }>(
    "get",
    "/api/v1/admin/database/tables",
    { params }
  );
};

export const getDatabaseColumns = (table: string, database?: string) => {
  return http.request<{ columns: DatabaseColumn[] }>(
    "get",
    `/api/v1/admin/database/tables/${encodeURIComponent(table)}/columns`,
    { params: { database } }
  );
};

export const getSiteAnnouncements = (
  params?: PageParams & { status?: string }
) => {
  return http.request<PagedResult<"announcements", SiteAnnouncement>>(
    "get",
    "/api/v1/admin/site/announcements",
    { params }
  );
};

export const saveSiteAnnouncement = (
  data: Partial<SiteAnnouncement>,
  id?: number
) => {
  return http.request<{ announcement: SiteAnnouncement }>(
    id ? "put" : "post",
    id
      ? `/api/v1/admin/site/announcements/${id}`
      : "/api/v1/admin/site/announcements",
    { data }
  );
};

export const deleteSiteAnnouncement = (id: number) => {
  return http.request<void>(
    "delete",
    `/api/v1/admin/site/announcements/${id}`
  );
};

export const getSiteBanners = (params?: PageParams & { status?: string }) => {
  return http.request<PagedResult<"banners", SiteBanner>>(
    "get",
    "/api/v1/admin/site/banners",
    { params }
  );
};

export const saveSiteBanner = (data: Partial<SiteBanner>, id?: number) => {
  return http.request<{ banner: SiteBanner }>(
    id ? "put" : "post",
    id ? `/api/v1/admin/site/banners/${id}` : "/api/v1/admin/site/banners",
    { data }
  );
};

export const deleteSiteBanner = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/site/banners/${id}`);
};

export const getSiteResources = (params?: PageParams & { status?: string }) => {
  return http.request<PagedResult<"resources", SiteResource>>(
    "get",
    "/api/v1/admin/site/resources",
    { params }
  );
};

export const saveSiteResource = (data: Partial<SiteResource>, id?: number) => {
  return http.request<{ resource: SiteResource }>(
    id ? "put" : "post",
    id ? `/api/v1/admin/site/resources/${id}` : "/api/v1/admin/site/resources",
    { data }
  );
};

export const deleteSiteResource = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/site/resources/${id}`);
};

export const getSiteTechStacks = (
  params?: PageParams & { status?: string }
) => {
  return http.request<PagedResult<"tech_stacks", SiteTechStack>>(
    "get",
    "/api/v1/admin/site/tech-stacks",
    { params }
  );
};

export const saveSiteTechStack = (
  data: Partial<SiteTechStack>,
  id?: number
) => {
  return http.request<{ tech_stack: SiteTechStack }>(
    id ? "put" : "post",
    id
      ? `/api/v1/admin/site/tech-stacks/${id}`
      : "/api/v1/admin/site/tech-stacks",
    { data }
  );
};

export const deleteSiteTechStack = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/site/tech-stacks/${id}`);
};

export const getSiteProjects = (params?: PageParams & { status?: string }) => {
  return http.request<PagedResult<"projects", SiteProject>>(
    "get",
    "/api/v1/admin/site/projects",
    { params }
  );
};

export const saveSiteProject = (data: Partial<SiteProject>, id?: number) => {
  return http.request<{ project: SiteProject }>(
    id ? "put" : "post",
    id ? `/api/v1/admin/site/projects/${id}` : "/api/v1/admin/site/projects",
    { data }
  );
};

export const deleteSiteProject = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/site/projects/${id}`);
};

export const getSiteTimelineEvents = (
  params?: PageParams & { status?: string }
) => {
  return http.request<PagedResult<"timeline", SiteTimelineEvent>>(
    "get",
    "/api/v1/admin/site/timeline",
    { params }
  );
};

export const saveSiteTimelineEvent = (
  data: Partial<SiteTimelineEvent>,
  id?: number
) => {
  return http.request<{ timeline_event: SiteTimelineEvent }>(
    id ? "put" : "post",
    id ? `/api/v1/admin/site/timeline/${id}` : "/api/v1/admin/site/timeline",
    { data }
  );
};

export const deleteSiteTimelineEvent = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/site/timeline/${id}`);
};

export const getSiteMessages = (params?: PageParams & { status?: string }) => {
  return http.request<PagedResult<"messages", SiteMessage>>(
    "get",
    "/api/v1/admin/site/messages",
    { params }
  );
};

export const saveSiteMessage = (data: Partial<SiteMessage>, id: number) => {
  return http.request<{ message: SiteMessage }>(
    "put",
    `/api/v1/admin/site/messages/${id}`,
    { data }
  );
};

export const deleteSiteMessage = (id: number) => {
  return http.request<void>("delete", `/api/v1/admin/site/messages/${id}`);
};

export const getSiteAnalytics = () => {
  return http.request<{ analytics: SiteAnalytics }>(
    "get",
    "/api/v1/admin/site/analytics"
  );
};

export const uploadSiteAsset = (file: File) => {
  const formData = new FormData();
  formData.append("file", file);
  return http.request<{ url: string }>("post", "/api/v1/admin/site/upload", {
    data: formData,
    headers: {
      "Content-Type": "multipart/form-data"
    }
  });
};
