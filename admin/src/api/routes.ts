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
  const ragChildren: Array<any> = [];

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

  if (hasAdminAccess && permissions.includes("announcements:read")) {
    children.push({
      path: "/go-admin/announcements",
      component: "go-admin/announcements/index",
      name: "GoAdminAnnouncements",
      meta: {
        title: "后台公告",
        icon: "ri:megaphone-line",
        auths: ["announcements:read", "announcements:write"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("ai:assistant")) {
    ragChildren.push({
      path: "/rag/ai-assistant",
      component: "rag/ai-assistant/index",
      name: "RAGAIAssistant",
      meta: {
        title: "AI 助手",
        icon: "ri:robot-2-line",
        auths: ["ai:assistant"]
      }
    });
    ragChildren.push({
      path: "/rag/documents",
      component: "rag/documents/index",
      name: "RAGDocuments",
      meta: {
        title: "文档管理",
        icon: "ri:file-upload-line",
        auths: ["ai:assistant"]
      }
    });
    ragChildren.push({
      path: "/rag/index",
      component: "rag/rag-index/index",
      name: "RAGIndex",
      meta: {
        title: "RAG 索引管理",
        icon: "ri:database-2-line",
        auths: ["ai:assistant"]
      }
    });
    ragChildren.push({
      path: "/rag/tuning",
      component: "rag/tuning/index",
      name: "RAGTuning",
      meta: {
        title: "调参中心",
        icon: "ri:equalizer-line",
        auths: ["ai:assistant"]
      }
    });
    ragChildren.push({
      path: "/rag/evals",
      component: "rag/evals/index",
      name: "RAGEvals",
      meta: {
        title: "评测中心",
        icon: "ri:flask-line",
        auths: ["ai:assistant"]
      }
    });
    ragChildren.push({
      path: "/rag/analytics",
      component: "rag/analytics/index",
      name: "RAGAnalytics",
      meta: {
        title: "命中分析",
        icon: "ri:line-chart-line",
        auths: ["ai:assistant"]
      }
    });
    ragChildren.push({
      path: "/rag/feedback",
      component: "rag/feedback/index",
      name: "RAGFeedback",
      meta: {
        title: "反馈处理",
        icon: "ri:feedback-line",
        auths: ["ai:assistant"]
      }
    });
  }

  if (hasAdminAccess && permissions.includes("ai:models:read")) {
    ragChildren.push({
      path: "/rag/ai-models",
      component: "rag/ai-models/index",
      name: "RAGAIModels",
      meta: {
        title: "大模型配置",
        icon: "ri:brain-line",
        auths: ["ai:models:read", "ai:models:write"]
      }
    });
    ragChildren.push({
      path: "/rag/ai-call-logs",
      component: "rag/ai-call-logs/index",
      name: "RAGAIModelCallLogs",
      meta: {
        title: "调用日志",
        icon: "ri:file-search-line",
        auths: ["ai:models:read"]
      }
    });
  }

  if (hasAdminAccess) {
    toolChildren.push({
      path: "/system-tools/settings",
      component: "system-tools/settings/index",
      name: "SystemToolsSettings",
      meta: {
        title: "系统配置中心",
        icon: "ri:settings-3-line",
        auths: ["admin:access"]
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
        rank: 13
      },
      children: toolChildren
    });
  }

  if (ragChildren.length > 0) {
    routes.push({
      path: "/rag",
      name: "RAG",
      meta: {
        title: "RAG",
        icon: "ri:mind-map",
        rank: 12
      },
      children: ragChildren
    });
  }

  if (hasAdminAccess && permissions.includes("site:read")) {
    routes.push({
      path: "/site-admin",
      name: "SiteAdmin",
      meta: {
        title: "官网管理",
        icon: "ri:global-line",
        rank: 14
      },
      children: [
        {
          path: "/site-admin/operations",
          component: "site-admin/operations/index",
          name: "SiteAdminOperations",
          meta: {
            title: "运营仪表盘",
            icon: "ri:dashboard-3-line",
            auths: ["site:read"]
          }
        },
        {
          path: "/site-admin/projects",
          component: "site-admin/projects/index",
          name: "SiteAdminProjects",
          meta: {
            title: "项目作品",
            icon: "ri:briefcase-4-line",
            auths: ["site:read", "site:write"]
          }
        },
        {
          path: "/site-admin/content",
          component: "site-admin/content/index",
          name: "SiteAdminContent",
          meta: {
            title: "内容发布",
            icon: "ri:layout-masonry-line",
            auths: ["site:read", "site:write"]
          }
        }
      ]
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
