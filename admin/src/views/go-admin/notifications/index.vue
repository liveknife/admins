<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  getNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  type Notification
} from "@/api/admin";
import { message } from "@/utils/message";
import RePagination from "@/components/RePagination";

defineOptions({ name: "GoAdminNotifications" });

const loading = ref(false);
const notices = ref<Notification[]>([]);
const readStatus = ref("");
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
});

const unreadCount = computed(() => notices.value.filter(item => !item.is_read).length);

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getNotifications({
      page: pagination.page,
      page_size: pagination.pageSize,
      read_status: readStatus.value || undefined
    });
    notices.value = res.notifications ?? [];
    pagination.total = res.total ?? 0;
  } catch {
    message("通知加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const handleFilter = () => {
  pagination.page = 1;
  loadData();
};

const setRead = async (row: Notification) => {
  if (row.is_read) return;
  await markNotificationRead(row.id);
  message("已标记为已读", { type: "success" });
  await loadData();
};

const setAllRead = async () => {
  await markAllNotificationsRead();
  message("全部通知已读", { type: "success" });
  await loadData();
};

const tagType = (type: string) => {
  if (type === "success") return "success";
  if (type === "warning") return "warning";
  if (type === "danger") return "danger";
  return "info";
};

onMounted(loadData);
</script>

<template>
  <div class="notice-page">
    <div class="page-head">
      <div>
        <h2>通知中心</h2>
        <p>集中查看系统通知、提醒和待处理消息。</p>
      </div>
      <div class="head-actions">
        <el-tag type="danger" effect="light">本页未读 {{ unreadCount }}</el-tag>
        <el-button type="primary" @click="setAllRead">全部已读</el-button>
      </div>
    </div>

    <div class="filter-bar">
      <el-segmented
        v-model="readStatus"
        :options="[
          { label: '全部', value: '' },
          { label: '未读', value: 'unread' },
          { label: '已读', value: 'read' }
        ]"
        @change="handleFilter"
      />
      <el-button @click="loadData">刷新</el-button>
    </div>

    <div class="notice-list" v-loading="loading">
      <div v-for="item in notices" :key="item.id" class="notice-item" :class="{ unread: !item.is_read }">
        <div class="notice-main">
          <div class="notice-title">
            <el-tag :type="tagType(item.type)" size="small">{{ item.type || "info" }}</el-tag>
            <span>{{ item.title }}</span>
            <el-badge v-if="!item.is_read" is-dot />
          </div>
          <div class="notice-content">{{ item.content }}</div>
          <div class="notice-time">{{ dayjs(item.created_at).format("YYYY-MM-DD HH:mm:ss") }}</div>
        </div>
        <el-button v-if="!item.is_read" link type="primary" @click="setRead(item)">标记已读</el-button>
        <el-tag v-else type="info">已读</el-tag>
      </div>
      <el-empty v-if="!loading && notices.length === 0" description="暂无通知" />
      <RePagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :disabled="loading"
        @change="loadData"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.notice-page {
  padding: 24px;
}
.page-head,
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
.head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.notice-list {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
  overflow: hidden;
}
.notice-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--app-border-soft);
}
.notice-item.unread {
  background: var(--app-surface-soft);
}
.notice-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
}
.notice-content {
  margin-top: 8px;
  color: var(--app-text-secondary);
}
.notice-time {
  margin-top: 8px;
  color: var(--app-text-muted);
  font-size: 12px;
}
@media (max-width: 768px) {
  .page-head,
  .filter-bar,
  .notice-item {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
