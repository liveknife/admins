<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { useI18n } from "@/i18n";
import {
  getOperationLogs,
  type OperationLog
} from "@/api/admin";
import { useUserStoreHook } from "@/store/modules/user";
import RePagination from "@/components/RePagination";

const loading = ref(false);
const logs = ref<OperationLog[]>([]);
const pagination = reactive({
  page: 1,
  pageSize: 15,
  total: 0
});

// 筛选
const actionFilter = ref("");
const searchKeyword = ref("");

// 详情弹窗
const detailDialogVisible = ref(false);
const currentLog = ref<OperationLog>();

const userStore = useUserStoreHook();
const { t } = useI18n();

// ── 操作类型选项 ──
const actionOptions = [
  { label: t("logs.allActions"), value: "" },
  { label: t("logs.actionLogin"), value: "login" },
  { label: t("logs.actionCreate"), value: "create" },
  { label: t("logs.actionEdit"), value: "update" },
  { label: t("logs.actionDelete"), value: "delete" },
  { label: t("logs.actionReactivate"), value: "reactivate" },
  { label: t("logs.actionReset"), value: "reset" }
];

// ── 操作类型 Tag 颜色映射 ──
type TagType = "primary" | "success" | "warning" | "danger" | "info";
const getActionTagType = (action: string): TagType => {
  const lowerAction = action.toLowerCase();
  if (lowerAction.includes("login") || lowerAction.includes("登")) return "success";
  if (lowerAction.includes("create") || lowerAction.includes("新建") || lowerAction.includes("创建")) return "primary";
  if (
    lowerAction.includes("update") ||
    lowerAction.includes("edit") ||
    lowerAction.includes("修改") ||
    lowerAction.includes("编辑") ||
    lowerAction.includes("重置") ||
    lowerAction.includes("恢复") ||
    lowerAction.includes("reactivate") ||
    lowerAction.includes("reset")
  ) return "warning";
  if (lowerAction.includes("delete") || lowerAction.includes("注销") || lowerAction.includes("停用") || lowerAction.includes("删除")) return "danger";
  return "info";
};

// ── 格式化时间 ──
const formatDateTime = (value: string) => {
  if (!value) return "-";
  const d = new Date(value);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
};

// ── 加载日志数据 ──
const loadData = async () => {
  loading.value = true;
  try {
    const params: Record<string, any> = {
      page: pagination.page,
      page_size: pagination.pageSize
    };
    if (actionFilter.value) params.action = actionFilter.value;
    if (searchKeyword.value) params.keyword = searchKeyword.value;

    const res = await getOperationLogs(params);
    logs.value = res.logs ?? [];
    pagination.total = res.total ?? 0;
  } catch (error) {
    message(t("logs.loadFailed"), { type: "error" });
  } finally {
    loading.value = false;
  }
};

// ── 查看详情 ──
const openDetailDialog = (row: OperationLog) => {
  currentLog.value = row;
  detailDialogVisible.value = true;
};

// ── 搜索 / 筛选变化时重置页码 ──
const handleFilterChange = () => {
  pagination.page = 1;
  loadData();
};

// ── 分页切换 ──
const handlePageChange = () => {
  loadData();
};

onMounted(loadData);
</script>

<template>
  <div class="page-container">
    <!-- 头部工具栏 -->
    <div class="page-header">
      <div class="page-header-left">
        <h2 class="page-title">{{ t("logs.title") }}</h2>
        <span class="page-badge">/api/v1/admin/operation-logs</span>
      </div>
      <el-space>
        <el-button :loading="loading" @click="loadData">
          {{ t("common.refresh") }}
        </el-button>
      </el-space>
    </div>

    <!-- 筛选栏 + 表格 -->
    <div class="table-card">
      <!-- 筛选栏 -->
      <div class="filter-bar">
        <el-select
          v-model="actionFilter"
          :placeholder="t('logs.filterAction')"
          clearable
          class="filter-select"
          @change="handleFilterChange"
        >
          <el-option
            v-for="opt in actionOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
        <el-input
          v-model="searchKeyword"
          :placeholder="t('logs.searchPlaceholder')"
          clearable
          class="filter-search"
          :prefix-icon="'Search'"
          @keyup.enter="handleFilterChange"
          @clear="handleFilterChange"
        >
          <template #append>
            <el-button :icon="'Search'" @click="handleFilterChange" />
          </template>
        </el-input>
      </div>

      <!-- 日志表格 -->
      <el-table
        v-loading="loading"
        :data="logs"
        stripe
        class="admin-table"
        row-key="id"
        @row-click="openDetailDialog"
      >
        <el-table-column prop="id" :label="t('common.id')" width="76" align="center" />

        <el-table-column prop="username" :label="t('logs.operator')" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="user-avatar">{{ row.username?.slice(0, 1)?.toUpperCase() }}</span>
              <span class="user-name">{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column prop="action" :label="t('logs.actionType')" width="120">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="getActionTagType(row.action)"
              effect="light"
              round
            >
              {{ row.action }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="resource" :label="t('logs.module')" min-width="120">
          <template #default="{ row }">
            <span class="muted-text">{{ row.resource || "-" }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="detail" :label="t('logs.detail')" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span>{{ row.detail || "-" }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="ip" :label="t('logs.ipAddress')" width="150">
          <template #default="{ row }">
            <span class="ip-text">{{ row.ip || "-" }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" :label="t('admin.createdAt')" width="175">
          <template #default="{ row }">
            <span class="muted-text">{{ formatDateTime(row.created_at) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <RePagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :disabled="loading"
        @change="handlePageChange"
      />
    </div>

    <!-- ═══ 日志详情弹窗 ═══ -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="t('logs.logDetail')"
      width="580px"
    >
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="t('common.id')">
          {{ currentLog?.id }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('logs.operator')">
          <div class="user-cell">
            <span class="user-avatar">{{ currentLog?.username?.slice(0, 1)?.toUpperCase() }}</span>
            <span class="user-name">{{ currentLog?.username }}</span>
          </div>
        </el-descriptions-item>
        <el-descriptions-item :label="t('logs.actionType')">
          <el-tag
            size="small"
            :type="getActionTagType(currentLog?.action ?? '')"
            effect="light"
            round
          >
            {{ currentLog?.action }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="t('logs.module')">
          {{ currentLog?.resource || "-" }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('logs.detail')">
          <div class="detail-content">{{ currentLog?.detail || "-" }}</div>
        </el-descriptions-item>
        <el-descriptions-item :label="t('logs.ipAddress')">
          <span class="ip-text">{{ currentLog?.ip || "-" }}</span>
        </el-descriptions-item>
        <el-descriptions-item :label="t('logs.userAgent')">
          <div class="ua-content">{{ currentLog?.user_agent || "-" }}</div>
        </el-descriptions-item>
        <el-descriptions-item :label="t('admin.createdAt')">
          {{ formatDateTime(currentLog?.created_at ?? "") }}
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="detailDialogVisible = false">{{ t("common.close") }}</el-button>
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

/* ═══ 筛选栏 ═══ */
.filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--app-border-soft);
  flex-wrap: wrap;
}

.filter-select {
  width: 160px;
}

.filter-search {
  width: 300px;
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
    height: 58px;
    cursor: pointer;
  }

  :deep(.el-table__cell) {
    border-color: var(--app-border-soft);
  }
}

.user-cell {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 10px;
}

.user-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 30px;
  width: 30px;
  height: 30px;
  color: var(--app-primary);
  font-size: 13px;
  font-weight: 700;
  background: linear-gradient(135deg, var(--app-bg-soft) 0%, #e8fff7 100%);
  border: 1px solid var(--app-border);
  border-radius: 50%;
}

.user-name {
  overflow: hidden;
  color: var(--app-text);
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.muted-text {
  color: var(--app-text-muted);
}

.ip-text {
  color: var(--app-text-secondary);
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12.5px;
}

.detail-content,
.ua-content {
  word-break: break-all;
  line-height: 1.5;
  max-height: 200px;
  overflow-y: auto;
  font-size: 13px;
  color: var(--app-text-secondary);
}

/* 响应式适配 */
@media (max-width: 768px) {
  .page-container {
    padding: 16px;
  }

  .filter-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .filter-select,
  .filter-search {
    width: 100%;
  }
}
</style>
