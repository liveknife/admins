<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import dayjs from "dayjs";
import BellIcon from "~icons/ep/bell";
import {
  getNotifications,
  getUnreadNotificationCount,
  markNotificationRead,
  type Notification
} from "@/api/admin";

const TEXT = {
  title: "\u901a\u77e5\u4e2d\u5fc3",
  all: "\u67e5\u770b\u5168\u90e8",
  empty: "\u6682\u65e0\u672a\u8bfb\u901a\u77e5",
  footer: "\u8fdb\u5165\u901a\u77e5\u4e2d\u5fc3",
  noUnread: "\u6682\u65e0\u672a\u8bfb\u6d88\u606f",
  unreadPrefix: "\u8fd8\u6709",
  unreadSuffix: "\u6761\u672a\u8bfb\u6d88\u606f",
  typeSuccess: "\u6210\u529f",
  typeWarning: "\u63d0\u9192",
  typeDanger: "\u91cd\u8981",
  typeInfo: "\u901a\u77e5"
};

const router = useRouter();
const loading = ref(false);
const unread = ref(0);
const notices = ref<Notification[]>([]);

const badgeValue = computed(() => (unread.value > 0 ? unread.value : ""));
const previewList = computed(() => notices.value.slice(0, 5));
const subtitle = computed(() =>
  unread.value > 0
    ? `${TEXT.unreadPrefix} ${unread.value} ${TEXT.unreadSuffix}`
    : TEXT.noUnread
);

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

const setRead = async (item: Notification) => {
  await markNotificationRead(item.id);
  await loadNotices();
};

const goCenter = () => {
  router.push("/go-admin/notifications");
};

const getNoticeTypeText = (type: string) => {
  if (type === "success") return TEXT.typeSuccess;
  if (type === "warning") return TEXT.typeWarning;
  if (type === "danger") return TEXT.typeDanger;
  return TEXT.typeInfo;
};

const getNoticeClass = (type: string) => {
  if (type === "success") return "is-success";
  if (type === "warning") return "is-warning";
  if (type === "danger") return "is-danger";
  return "is-info";
};

onMounted(loadNotices);
</script>

<template>
  <el-dropdown
    trigger="click"
    placement="bottom-end"
    popper-class="notice-popper"
    @visible-change="visible => visible && loadNotices()"
  >
    <span class="dropdown-badge navbar-bg-hover select-none">
      <el-badge :value="badgeValue" :max="99">
        <span class="header-notice-icon">
          <IconifyIconOffline :icon="BellIcon" />
        </span>
      </el-badge>
    </span>
    <template #dropdown>
      <div class="notice-dropdown" v-loading="loading">
        <div class="notice-head">
          <div>
            <div class="notice-head-title">{{ TEXT.title }}</div>
            <div class="notice-head-subtitle">{{ subtitle }}</div>
          </div>
          <el-button class="notice-head-action" link type="primary" @click.stop="goCenter">
            {{ TEXT.all }}
          </el-button>
        </div>

        <el-scrollbar max-height="340px">
          <div v-if="previewList.length" class="notice-list">
            <button
              v-for="item in previewList"
              :key="item.id"
              class="notice-item"
              type="button"
              @click="setRead(item)"
            >
              <span class="notice-status" :class="getNoticeClass(item.type)" />
              <span class="notice-body">
                <span class="notice-row">
                  <span class="notice-title">{{ item.title }}</span>
                  <span class="notice-type" :class="getNoticeClass(item.type)">
                    {{ getNoticeTypeText(item.type) }}
                  </span>
                </span>
                <span class="notice-content">{{ item.content }}</span>
                <span class="notice-time">{{ dayjs(item.created_at).format("MM-DD HH:mm") }}</span>
              </span>
            </button>
            <button class="notice-footer" type="button" @click.stop="goCenter">
              {{ TEXT.footer }}
            </button>
          </div>
          <el-empty v-else :description="TEXT.empty" :image-size="72" />
        </el-scrollbar>
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

.notice-dropdown {
  width: 380px;
  padding: 0;
  overflow: hidden;
  background: #fff;
  border: 1px solid #e7ebf3;
  border-radius: 8px;
  box-shadow: 0 18px 45px rgb(15 23 42 / 14%);
}

.notice-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 18px 14px;
  background: linear-gradient(180deg, #f8fbff 0%, #fff 100%);
  border-bottom: 1px solid #edf0f5;
}

.notice-head-title {
  color: #111827;
  font-size: 15px;
  font-weight: 700;
  line-height: 22px;
}

.notice-head-subtitle {
  margin-top: 2px;
  color: #7b8494;
  font-size: 12px;
  line-height: 18px;
}

.notice-head-action {
  height: 28px;
  padding: 0 4px;
  font-weight: 600;
}

.notice-list {
  padding: 8px;
}

.notice-item {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr);
  gap: 10px;
  width: 100%;
  padding: 12px;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 8px;
  transition:
    background 0.15s,
    transform 0.15s;

  &:hover {
    background: #f8fbff;
    transform: translateY(-1px);
  }
}

.notice-status {
  width: 8px;
  height: 8px;
  margin-top: 7px;
  border-radius: 50%;
  background: #2f6bff;
}

.notice-status.is-success {
  background: #16a34a;
}

.notice-status.is-warning {
  background: #d97706;
}

.notice-status.is-danger {
  background: #dc2626;
}

.notice-body {
  min-width: 0;
}

.notice-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.notice-title {
  min-width: 0;
  overflow: hidden;
  color: #111827;
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notice-type {
  flex: 0 0 auto;
  height: 22px;
  padding: 0 8px;
  color: #2563eb;
  font-size: 12px;
  line-height: 22px;
  background: #eff6ff;
  border-radius: 999px;
}

.notice-type.is-success {
  color: #15803d;
  background: #f0fdf4;
}

.notice-type.is-warning {
  color: #b45309;
  background: #fffbeb;
}

.notice-type.is-danger {
  color: #b91c1c;
  background: #fef2f2;
}

.notice-content {
  display: -webkit-box;
  margin-top: 4px;
  overflow: hidden;
  color: #6b7280;
  font-size: 12px;
  line-height: 18px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.notice-time {
  display: block;
  margin-top: 7px;
  color: #9ca3af;
  font-size: 12px;
  line-height: 18px;
}

.notice-footer {
  width: 100%;
  height: 36px;
  margin-top: 2px;
  color: #2f6bff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  background: #f6f8fb;
  border: 0;
  border-radius: 8px;
  transition: background 0.15s;

  &:hover {
    background: #edf3ff;
  }
}

:global(.notice-popper.el-popper) {
  border: 0 !important;
  border-radius: 8px !important;
  box-shadow: none !important;
}

:global(.notice-popper .el-popper__arrow::before) {
  background: #f8fbff !important;
  border-color: #e7ebf3 !important;
}
</style>
