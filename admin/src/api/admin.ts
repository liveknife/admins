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

export type AIAssistantResult = {
  question: string;
  answer: string;
  insights: string[];
  rows: Array<Record<string, any>>;
  metrics: Record<string, number>;
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

export const askAIAssistant = (question: string) => {
  return http.request<{ result: AIAssistantResult }>(
    "post",
    "/api/v1/admin/ai/ask",
    { data: { question } }
  );
};

export const getSystemHealth = () => {
  return http.request<{ health: SystemHealth }>(
    "get",
    "/api/v1/admin/health"
  );
};
