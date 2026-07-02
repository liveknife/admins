<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  getAnnouncements,
  createAnnouncement,
  updateAnnouncement,
  deleteAnnouncement,
  type AdminAnnouncement
} from "@/api/admin";
import { message } from "@/utils/message";
import { useI18n } from "@/i18n";
import { useUserStoreHook } from "@/store/modules/user";
import RePagination from "@/components/RePagination";

defineOptions({ name: "GoAdminAnnouncements" });

const { t } = useI18n();
const userStore = useUserStoreHook();

const canWrite = computed(() =>
  userStore.permissions.includes("announcements:write")
);

const loading = ref(false);
const notices = ref<AdminAnnouncement[]>([]);
const pagination = reactive({ page: 1, pageSize: 10, total: 0 });

const dialogVisible = ref(false);
const dialogTitle = ref("");
const form = reactive({ title: "", content: "", type: "info", is_active: true });
const editingId = ref<number>();

const tagType = (type: string) => {
  if (type === "success") return "success";
  if (type === "warning") return "warning";
  if (type === "danger") return "danger";
  return "info";
};

const typeLabel = (type: string) => {
  if (type === "success") return t("announce.typeSuccess");
  if (type === "warning") return t("announce.typeWarning");
  if (type === "danger") return t("announce.typeDanger");
  return t("announce.typeInfo");
};

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getAnnouncements({
      page: pagination.page,
      page_size: pagination.pageSize
    });
    notices.value = res.announcements ?? [];
    pagination.total = res.total ?? 0;
  } catch {
    message(t("announce.loadFailed"), { type: "error" });
  } finally {
    loading.value = false;
  }
};

const resetForm = () => {
  form.title = "";
  form.content = "";
  form.type = "info";
  form.is_active = true;
  editingId.value = undefined;
};

const openCreate = () => {
  resetForm();
  dialogTitle.value = t("announce.create");
  dialogVisible.value = true;
};

const openEdit = (row: AdminAnnouncement) => {
  form.title = row.title;
  form.content = row.content;
  form.type = row.type;
  form.is_active = row.is_active;
  editingId.value = row.id;
  dialogTitle.value = t("announce.edit");
  dialogVisible.value = true;
};

const submitSave = async () => {
  if (!form.title.trim()) {
    message(t("announce.titleRequired"), { type: "warning" });
    return;
  }
  try {
    if (editingId.value) {
      await updateAnnouncement(editingId.value, {
        title: form.title,
        content: form.content,
        type: form.type,
        is_active: form.is_active
      });
      message(t("announce.updated"), { type: "success" });
    } else {
      await createAnnouncement({
        title: form.title,
        content: form.content,
        type: form.type,
        is_active: form.is_active
      });
      message(t("announce.created"), { type: "success" });
    }
    dialogVisible.value = false;
    await loadData();
  } catch {
    /* handled by interceptor */
  }
};

const removeItem = async (row: AdminAnnouncement) => {
  try {
    await deleteAnnouncement(row.id);
    message(t("announce.deleted"), { type: "success" });
    await loadData();
  } catch {
    /* handled by interceptor */
  }
};

const toggleActive = async (row: AdminAnnouncement) => {
  try {
    await updateAnnouncement(row.id, {
      title: row.title,
      content: row.content,
      type: row.type,
      is_active: !row.is_active
    });
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
        <h2 class="page-title">{{ t("announce.title") }}</h2>
        <span class="page-badge">/api/v1/admin/announcements</span>
      </div>
      <el-space>
        <el-button v-if="canWrite" type="primary" @click="openCreate">
          + {{ t("announce.create") }}
        </el-button>
        <el-button :loading="loading" @click="loadData">
          {{ t("common.refresh") }}
        </el-button>
      </el-space>
    </div>

    <div class="table-card">
      <el-table v-loading="loading" :data="notices" stripe class="admin-table" row-key="id">
        <el-table-column :label="t('announce.type')" width="80">
          <template #default="{ row }">
            <el-tag :type="tagType(row.type)" size="small">{{ typeLabel(row.type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="title" :label="t('common.title')" min-width="180" />
        <el-table-column prop="content" :label="t('announce.content')" min-width="280" show-overflow-tooltip />
        <el-table-column :label="t('announce.status')" width="90">
          <template #default="{ row }">
            <el-switch
              :model-value="row.is_active"
              :disabled="!canWrite"
              :active-text="t('announce.enable')"
              :inactive-text="t('announce.disable')"
              inline-prompt
              @change="toggleActive(row)"
            />
          </template>
        </el-table-column>
        <el-table-column :label="t('announce.createdAt')" width="170">
          <template #default="{ row }">
            {{ dayjs(row.created_at).format("YYYY-MM-DD HH:mm") }}
          </template>
        </el-table-column>
        <el-table-column v-if="canWrite" :label="t('common.operation')" fixed="right" width="150">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">
              {{ t("common.edit") }}
            </el-button>
            <el-popconfirm :title="t('announce.deleteConfirm')" @confirm="removeItem(row)">
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

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="580px"
      destroy-on-close
    >
      <el-form :model="form" label-width="80px">
        <el-form-item :label="t('announce.type')">
          <el-select v-model="form.type">
            <el-option :label="t('announce.typeInfo')" value="info" />
            <el-option :label="t('announce.typeSuccess')" value="success" />
            <el-option :label="t('announce.typeWarning')" value="warning" />
            <el-option :label="t('announce.typeDanger')" value="danger" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('common.title')" required>
          <el-input v-model="form.title" maxlength="160" show-word-limit />
        </el-form-item>
        <el-form-item :label="t('announce.content')">
          <el-input v-model="form.content" type="textarea" :rows="5" />
        </el-form-item>
        <el-form-item :label="t('announce.status')">
          <el-switch v-model="form.is_active" :active-text="t('announce.enable')" :inactive-text="t('announce.disable')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" @click="submitSave">{{ editingId ? t("common.save") : t("announce.create") }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style lang="scss" scoped>
.page-container { padding: 24px 28px; }
.page-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 20px; }
.page-header-left { display: flex; align-items: center; gap: 10px; }
.page-title { font-size: 17px; font-weight: 700; color: var(--app-text); margin: 0; }
.page-badge { display: inline-block; padding: 2px 8px; background: var(--app-bg-soft); color: var(--app-violet); font-size: 11.5px; font-weight: 500; border-radius: 4px; border: 1px solid #e0e7ff; font-family: "Menlo", "Monaco", monospace; }
.table-card { background: var(--app-surface); border-radius: 8px; border: 1px solid var(--app-border-soft); overflow: hidden; box-shadow: 0 10px 28px rgb(33 49 77 / 8%); }
.admin-table {
  --el-table-header-bg-color: var(--app-surface-muted);
  --el-table-header-text-color: var(--app-text-secondary);
  --el-table-row-hover-bg-color: var(--app-surface-soft);
  :deep(.el-table__header th) { height: 46px; font-weight: 700; background: var(--app-surface-muted) !important; }
  :deep(.el-table__row) { height: 56px; }
  :deep(.el-table__cell) { border-color: var(--app-border-soft); }
}
</style>
