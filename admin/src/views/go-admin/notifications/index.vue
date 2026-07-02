<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  getNotifications,
  createNotification,
  deleteNotification,
  markAllNotificationsRead,
  markNotificationRead,
  type Notification
} from "@/api/admin";
import { message } from "@/utils/message";
import { useI18n } from "@/i18n";
import { useUserStoreHook } from "@/store/modules/user";
import RePagination from "@/components/RePagination";

defineOptions({ name: "GoAdminNotifications" });

const { t } = useI18n();
const userStore = useUserStoreHook();

const canWrite = computed(() =>
  userStore.permissions.includes("notifications:write")
);

const loading = ref(false);
const notices = ref<Notification[]>([]);
const readStatus = ref("");
const pagination = reactive({ page: 1, pageSize: 10, total: 0 });

// 发送通知对话框
const dialogVisible = ref(false);
const form = reactive({ title: "", content: "", type: "info" });

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
    message(t("notif.loadFailed"), { type: "error" });
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
  loadData();
};

const setAllRead = async () => {
  await markAllNotificationsRead();
  message(t("notif.allRead"), { type: "success" });
  loadData();
};

const tagType = (type: string) => {
  if (type === "success") return "success";
  if (type === "warning") return "warning";
  if (type === "danger") return "danger";
  return "info";
};

const openCreate = () => {
  form.title = "";
  form.content = "";
  form.type = "info";
  dialogVisible.value = true;
};

const submitCreate = async () => {
  if (!form.title.trim()) {
    message(t("notif.titleRequired"), { type: "warning" });
    return;
  }
  try {
    await createNotification({
      title: form.title,
      content: form.content,
      type: form.type
    });
    message(t("notif.sent"), { type: "success" });
    dialogVisible.value = false;
    await loadData();
  } catch {
    message(t("notif.sendFailed"), { type: "error" });
  }
};

const removeNotification = async (row: Notification) => {
  try {
    await deleteNotification(row.id);
    message(t("notif.deleted"), { type: "success" });
    await loadData();
  } catch {
    /* handled by interceptor */
  }
};

onMounted(loadData);
</script>

<template>
  <div class="page-container">
    <div class="page-header">
      <div class="page-header-left">
        <h2 class="page-title">{{ t("notif.title") }}</h2>
        <span class="page-badge">/api/v1/admin/notifications</span>
      </div>
      <el-space>
        <el-button v-if="canWrite" type="primary" @click="openCreate">
          + {{ t("notif.send") }}
        </el-button>
        <el-button @click="setAllRead">{{ t("notif.markAllRead") }}</el-button>
      </el-space>
    </div>

    <div class="table-card toolbar-card">
      <div class="toolbar-inner">
        <el-segmented
          v-model="readStatus"
          :options="[
            { label: t('notif.all'), value: '' },
            { label: t('notif.unread'), value: 'unread' },
            { label: t('notif.read'), value: 'read' }
          ]"
          @change="handleFilter"
        />
        <el-button :loading="loading" @click="loadData">
          {{ t("common.refresh") }}
        </el-button>
      </div>
    </div>

    <div class="table-card">
      <el-table v-loading="loading" :data="notices" stripe class="admin-table" row-key="id">
        <el-table-column :label="t('notif.type')" width="90">
          <template #default="{ row }">
            <el-tag :type="tagType(row.type)" size="small">
              {{ row.type === "success" ? t("notif.typeSuccess") : row.type === "warning" ? t("notif.typeWarning") : row.type === "danger" ? t("notif.typeDanger") : t("notif.typeInfo") }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" :label="t('common.title')" min-width="180" />
        <el-table-column prop="content" :label="t('notif.content')" min-width="280" show-overflow-tooltip />
        <el-table-column :label="t('notif.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.is_read ? 'info' : 'danger'" size="small">
              {{ row.is_read ? t('notif.read') : t('notif.unread') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('notif.createdAt')" width="170">
          <template #default="{ row }">
            {{ dayjs(row.created_at).format("YYYY-MM-DD HH:mm") }}
          </template>
        </el-table-column>
        <el-table-column v-if="canWrite" :label="t('common.operation')" fixed="right" width="150">
          <template #default="{ row }">
            <el-button v-if="!row.is_read" link type="primary" @click="setRead(row)">
              {{ t("notif.markRead") }}
            </el-button>
            <el-popconfirm :title="t('notif.deleteConfirm')" @confirm="removeNotification(row)">
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
        :total="pagination.total"
        :disabled="loading"
        @change="loadData"
      />
    </div>

    <!-- 发送通知弹窗 -->
    <el-dialog
      v-model="dialogVisible"
      :title="t('notif.send')"
      width="560px"
      destroy-on-close
    >
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('notif.type')">
          <el-select v-model="form.type">
            <el-option :label="t('notif.typeSuccess')" value="success" />
            <el-option :label="t('notif.typeWarning')" value="warning" />
            <el-option :label="t('notif.typeDanger')" value="danger" />
            <el-option :label="t('notif.typeInfo')" value="info" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.title')" required>
          <el-input v-model="form.title" :placeholder="t('notif.titlePlaceholder')" maxlength="160" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('notif.content')">
          <el-input v-model="form.content" type="textarea" :rows="5" :placeholder="t('notif.contentPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" @click="submitCreate">{{ t("notif.send") }}</el-button>
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
  padding: 12px 16px;
  flex-wrap: wrap;
}

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
</style>
