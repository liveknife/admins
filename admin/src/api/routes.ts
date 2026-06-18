import { http } from "@/utils/http";
import type { GoUser } from "./user";

type Result = {
  success: boolean;
  data: Array<any>;
};

const buildAdminRoutes = (user: GoUser) => {
  const permissions = user.permissions ?? [];
  const hasAdminAccess = permissions.includes("admin:access");

  const routes: Array<any> = [];
  const children: Array<any> = [];
  const toolChildren: Array<any> = [];

  if (hasAdminAccess && permissions.includes("users:read")) {
    children.push({
      path: "/go-admin/users",
      component: "go-admin/users/index",
      name: "GoAdminUsers",
      meta: {
        title: "用户管理",
        icon: "ri:user-settings-line",
        auths: ["users:read", "users:write"]
      }
    });
  }

  if (
    hasAdminAccess &&
    (permissions.includes("roles:read") ||
      permissions.includes("permissions:read"))
  ) {
    children.push({
      path: "/go-admin/roles",
      component: "go-admin/roles/index",
      name: "GoAdminRoles",
      meta: {
        title: "角色权限",
        icon: "ri:shield-keyhole-line",
        auths: ["roles:read", "roles:write", "permissions:read"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("logs:read")) {
    children.push({
      path: "/go-admin/operation-logs",
      component: "go-admin/operation-logs/index",
      name: "GoAdminOperationLogs",
      meta: {
        title: "操作日志",
        icon: "ri:file-list-3-line",
        auths: ["logs:read"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("notifications:read")) {
    children.push({
      path: "/go-admin/notifications",
      component: "go-admin/notifications/index",
      name: "GoAdminNotifications",
      meta: {
        title: "通知中心",
        icon: "ri:notification-3-line",
        auths: ["notifications:read", "notifications:write"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("ai:assistant")) {
    toolChildren.push({
      path: "/system-tools/ai-assistant",
      component: "system-tools/ai-assistant/index",
      name: "SystemToolsAIAssistant",
      meta: {
        title: "AI 助手",
        icon: "ri:robot-2-line",
        auths: ["ai:assistant"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("health:read")) {
    toolChildren.push({
      path: "/system-tools/health",
      component: "system-tools/health/index",
      name: "SystemToolsHealth",
      meta: {
        title: "系统健康监控",
        icon: "ri:pulse-line",
        auths: ["health:read"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("database:read")) {
    toolChildren.push({
      path: "/system-tools/database",
      component: "system-tools/database/index",
      name: "SystemToolsDatabase",
      meta: {
        title: "数据库表结构",
        icon: "ri:database-2-line",
        auths: ["database:read"]
      }
    });
  }

  if (children.length > 0) {
    routes.push({
      path: "/go-admin",
      name: "GoAdmin",
      meta: {
        title: "系统管理",
        icon: "ri:server-line",
        rank: 10
      },
      children
    });
  }

  if (toolChildren.length > 0) {
    routes.push({
      path: "/system-tools",
      name: "SystemTools",
      meta: {
        title: "系统工具",
        icon: "ri:tools-line",
        rank: 12
      },
      children: toolChildren
    });
  }

  if (permissions.includes("messages:chat")) {
    routes.push({
      path: "/message",
      name: "MessageManage",
      meta: {
        title: "消息管理",
        icon: "ri:message-3-line",
        rank: 11
      },
      children: [
        {
          path: "/message/chat",
          component: "message/chat/index",
          name: "MessageChat",
          meta: {
            title: "聊天",
            icon: "ri:chat-3-line",
            showParent: true,
            keepAlive: false,
            auths: ["messages:chat"]
          }
        }
      ]
    });
  }

  return routes;
};

export const getAsyncRoutes = () => {
  return http
    .request<{ user: GoUser }>("get", "/api/v1/me")
    .then(({ user }) => {
      return {
        success: true,
        data: buildAdminRoutes(user)
      } satisfies Result;
    });
};
