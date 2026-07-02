<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import BellIcon from "~icons/ep/bell";
import {
  getNotifications,
  getUnreadNotificationCount,
  markNotificationRead,
  getPublicAnnouncements,
  type Notification,
  type AdminAnnouncement
} from "@/api/admin";
import { useUserStoreHook } from "@/store/modules/user";

const TEXT = {
  notice: "\u901a\u77e5",
  announce: "\u516c\u544a",
  noticeEmpty: "\u6682\u65e0\u672a\u8bfb\u901a\u77e5",
  announceEmpty: "\u6682\u65e0\u516c\u544a",
  allNotices: "\u67e5\u770b\u5168\u90e8\u901a\u77e5",
  allAnnounces: "\u67e5\u770b\u516c\u544a",
  noUnread: "\u6682\u65e0\u672a\u8bfb\u6d88\u606f",
  unreadPrefix: "\u8fd8\u6709",
  unreadSuffix: "\u6761\u672a\u8bfb"
};

type TabType = "notice" | "announce";

const router = useRouter();
const userStore = useUserStoreHook();
const activeTab = ref<TabType>("notice");
const loading = ref(false);
const unread = ref(0);
const notices = ref<Notification[]>([]);
const announcements = ref<AdminAnnouncement[]>([]);

const badgeValue = computed(() => (unread.value > 0 ? unread.value : ""));
// 是否有公告管理权限
const canManageAnnounce = computed(() =>
  userStore.permissions.includes("announcements:write")
);
const previewList = computed(() => notices.value.slice(0, 5));
const subtitle = computed(() =>
  unread.value > 0 ? `${TEXT.unreadPrefix} ${unread.value} ${TEXT.unreadSuffix}` : TEXT.noUnread
);

/* ── 数据加载 ── */
const loadNotices = async () => {
  loading.value = true;
  try {
    const [countRes, listRes] = await Promise.all([
      getUnreadNotificationCount(),
      getNotifications({ page: 1, page_size: 6, read_status: "unread" })
    ]);
    unread.value = countRes.count ?? 0;
    notices.value = listRes.notifications ?? [];
  } finally {
    loading.value = false;
  }
};

const loadAnnouncements = async () => {
  try {
    const res = await getPublicAnnouncements();
    announcements.value = res.announcements ?? [];
  } catch {
    announcements.value = [];
  }
};

watch(activeTab, tab => {
  if (tab === "announce" && announcements.value.length === 0) loadAnnouncements();
});

/* ── 操作 ── */
const setRead = async (item: Notification) => {
  await markNotificationRead(item.id);
  await loadNotices();
};

const goNoticeCenter = () => {
  router.push("/go-admin/notifications");
};

const goAnnounceCenter = () => {
  router.push("/go-admin/announcements");
};

/* ── 样式辅助 ── */
const getNoticeTypeText = (type: string) => {
  if (type === "success") return "\u6210\u529f";
  if (type === "warning") return "\u63d0\u9192";
  if (type === "danger") return "\u91cd\u8981";
  return "\u901a\u77e5";
};

const getNoticeClass = (type: string) => {
  if (type === "success") return "is-success";
  if (type === "warning") return "is-warning";
  if (type === "danger") return "is-danger";
  return "is-info";
};

const announceBadgeColor = (type: string) => {
  if (type === "success") return "var(--app-green)";
  if (type === "warning") return "#d97706";
  if (type === "danger") return "var(--app-red)";
  return "var(--app-primary)";
};

const announceBadgeBg = (type: string) => {
  if (type === "success") return "#f0fdf4";
  if (type === "warning") return "#fffbeb";
  if (type === "danger") return "#fef2f2";
  return "var(--app-bg-soft)";
};

onMounted(loadNotices);
</script>

<template>
  <el-dropdown
    trigger="click"
    placement="bottom-end"
    popper-class="message-popper"
    @visible-change="(visible: boolean) => visible && loadNotices()"
  >
    <span class="dropdown-badge navbar-bg-hover select-none">
      <el-badge :value="badgeValue" :max="99">
        <span class="header-notice-icon">
          <IconifyIconOffline :icon="BellIcon" />
        </span>
      </el-badge>
    </span>
    <template #dropdown>
      <div class="message-panel" v-loading="loading">
        <!-- Tab 切换栏 -->
        <div class="msg-tabs">
          <button
            class="msg-tab"
            :class="{ 'is-active': activeTab === 'notice' }"
            @click.stop="activeTab = 'notice'"
          >
            {{ TEXT.notice }}
            <span v-if="unread > 0" class="msg-tab-badge">{{ unread }}</span>
          </button>
          <button
            class="msg-tab"
            :class="{ 'is-active': activeTab === 'announce' }"
            @click.stop="activeTab = 'announce'"
          >
            {{ TEXT.announce }}
          </button>
          <!-- 底部滑块 -->
          <div
            class="msg-tab-slider"
            :style="{ transform: activeTab === 'notice' ? 'translateX(0)' : 'translateX(100%)' }"
          />
        </div>

        <!-- 通知列表 -->
        <template v-if="activeTab === 'notice'">
          <div v-if="previewList.length" class="msg-list">
            <button
              v-for="item in previewList"
              :key="item.id"
              class="msg-item notice-item"
              type="button"
              @click="setRead(item)"
            >
              <span class="msg-dot" :class="getNoticeClass(item.type)" />
              <span class="msg-body">
                <span class="msg-row">
                  <span class="msg-title">{{ item.title }}</span>
                  <span class="msg-tag" :class="getNoticeClass(item.type)">
                    {{ getNoticeTypeText(item.type) }}
                  </span>
                </span>
                <span class="msg-desc">{{ item.content }}</span>
                <span class="msg-time">{{ dayjs(item.created_at).format("MM-DD HH:mm") }}</span>
              </span>
            </button>
            <button class="msg-footer" type="button" @click.stop="goNoticeCenter">
              {{ TEXT.allNotices }} &rarr;
            </button>
          </div>
          <div v-else class="msg-empty">
            <IconifyIconOffline :icon="BellIcon" class="msg-empty-icon" />
            <p>{{ TEXT.noticeEmpty }}</p>
          </div>
        </template>

        <!-- 公告列表 -->
        <template v-else>
          <div v-if="announcements.length" class="msg-list">
            <div
              v-for="item in announcements.slice(0, 5)"
              :key="item.id"
              class="msg-item announce-item"
            >
              <span
                class="announce-tag"
                :style="{
                  color: announceBadgeColor(item.type),
                  background: announceBadgeBg(item.type)
                }"
              >
                {{ getNoticeTypeText(item.type) }}
              </span>
              <span class="msg-body">
                <span class="msg-title">{{ item.title }}</span>
                <span class="msg-desc">{{ item.content || "-" }}</span>
                <span class="msg-time">
                  {{ dayjs(item.updated_at || item.created_at).format("YYYY-MM-DD HH:mm") }}
                </span>
              </span>
            </div>
            <button v-if="canManageAnnounce" class="msg-footer" type="button" @click.stop="goAnnounceCenter">
              {{ TEXT.allAnnounces }} &rarr;
            </button>
          </div>
          <div v-else class="msg-empty">
            <p>{{ TEXT.announceEmpty }}</p>
          </div>
        </template>
      </div>
    </template>
  </el-dropdown>
</template>

<style lang="scss" scoped>
.dropdown-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 48px;
  cursor: pointer;

  .header-notice-icon {
    font-size: 18px;
  }
}

/* ── 面板容器 ── */
.message-panel {
  width: 400px;
  padding: 0;
  overflow: hidden;
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 12px;
  box-shadow:
    0 4px 24px rgba(0, 0, 0, 0.08),
    0 0 0 1px rgba(0, 0, 0, 0.02);
}

/* ── Tab 栏 ── */
.msg-tabs {
  position: relative;
  display: flex;
  gap: 0;
  padding: 4px 16px 0;
  background: var(--app-surface);
}

.msg-tab {
  position: relative;
  flex: 1;
  height: 40px;
  padding: 0;
  color: var(--app-text-muted);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  background: transparent;
  border: 0;
  transition: color 0.25s;

  &:hover {
    color: var(--app-text-secondary);
  }

  &.is-active {
    color: var(--app-text);
    font-weight: 600;
  }
}

.msg-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  margin-left: 6px;
  padding: 0 5px;
  font-size: 11px;
  font-weight: 700;
  line-height: 18px;
  color: var(--app-red);
  background: var(--app-red);
  border-radius: 9px;
  vertical-align: middle;

  /* 白色文字 */
  color: #fff;
}

.msg-tab-slider {
  position: absolute;
  bottom: 0;
  left: 0;
  width: 50%;
  height: 2px;
  background: var(--app-primary);
  border-radius: 1px;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

/* ── 列表区 ── */
.msg-list {
  max-height: 360px;
  overflow-y: auto;
  padding: 8px 12px 4px;

  &::-webkit-scrollbar {
    width: 4px;
  }
  &::-webkit-scrollbar-thumb {
    background: var(--app-border-soft);
    border-radius: 2px;
  }
}

/* ── 通用条目 ── */
.msg-item {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px;
  width: 100%;
  padding: 14px 12px;
  text-align: left;
  cursor: default;
  background: transparent;
  border: none;
  border-radius: 10px;
  transition: background 0.15s;

  &:hover {
    background: var(--app-surface-soft);
  }

  &.notice-item {
    cursor: pointer;

    &:hover {
      background: var(--app-surface-soft);
      transform: translateY(-1px);
    }
  }

  &.announce-item {
    cursor: default;
  }
}

.msg-dot {
  flex-shrink: 0;
  width: 8px;
  height: 8px;
  margin-top: 8px;
  border-radius: 50%;
  background: var(--app-primary);

  &.is-success {
    background: var(--app-green);
  }

  &.is-warning {
    background: #d97706;
  }

  &.is-danger {
    background: var(--app-red);
  }
}

.announce-tag {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 22px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 6px;
}

.msg-body {
  min-width: 0;
}

.msg-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.msg-title {
  overflow: hidden;
  color: var(--app-text);
  font-size: 13.5px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.msg-tag {
  flex-shrink: 0;
  height: 20px;
  padding: 0 8px;
  font-size: 11px;
  font-weight: 500;
  line-height: 20px;
  background: var(--app-bg-soft);
  border-radius: 999px;

  color: var(--app-primary);

  &.is-success {
    color: var(--app-green);
    background: #f0fdf4;
  }

  &.is-warning {
    color: #d97706;
    background: #fffbeb;
  }

  &.is-danger {
    color: var(--app-red);
    background: #fef2f2;
  }
}

.msg-desc {
  display: -webkit-box;
  margin-top: 4px;
  overflow: hidden;
  color: var(--app-text-secondary);
  font-size: 12.5px;
  line-height: 19px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.msg-time {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  color: var(--app-text-muted);
  font-size: 12px;
  line-height: 17px;
}

.status-active {
  display: inline-block;
  padding: 0 6px;
  font-size: 10.5px;
  font-weight: 600;
  color: var(--app-green);
  background: #f0fdf4;
  border-radius: 4px;
}

.status-inactive {
  display: inline-block;
  padding: 0 6px;
  font-size: 10.5px;
  font-weight: 600;
  color: var(--app-text-muted);
  background: var(--app-surface-muted);
  border-radius: 4px;
}

/* ── Footer 按钮 ── */
.msg-footer {
  width: calc(100% - 24px);
  height: 38px;
  margin: 4px 12px 10px;
  color: var(--app-primary);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--app-primary) 6%, transparent) 0%,
    color-mix(in srgb, var(--app-primary) 10%, transparent) 100%
  );
  border: 1px solid color-mix(in srgb, var(--app-primary) 15%, transparent);
  border-radius: 8px;
  transition:
    background 0.15s,
    border-color 0.15s;

  &:hover {
    background: linear-gradient(
      135deg,
      color-mix(in srgb, var(--app-primary) 10%, transparent) 0%,
      color-mix(in srgb, var(--app-primary) 15%, transparent) 100%
    );
    border-color: color-mix(in srgb, var(--app-primary) 30%, transparent);
  }
}

/* ── 空状态 ── */
.msg-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: var(--app-text-muted);

  p {
    margin-top: 12px;
    font-size: 13px;
  }
}

.msg-empty-icon {
  opacity: 0.35;
  font-size: 32px;
}
</style>

<style lang="scss">
/* 全局 popper 样式（不能 scoped） */
.message-popper.el-popper {
  border: 0 !important;
  border-radius: 12px !important;
  box-shadow: none !important;
  padding: 0 !important;
}

.message-popper .el-popper__arrow::before {
  background: var(--app-surface) !important;
  border-color: var(--app-border) !important;
}
</style>
