<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from "vue";
import type { FormInstance, FormRules } from "element-plus";
import { ElMessageBox, type ElTree } from "element-plus";
import { message } from "@/utils/message";
import { useI18n } from "@/i18n";
import {
  createAdminRole,
  deleteAdminRole,
  getAdminPermissions,
  getAdminRoles,
  getPermissionTree,
  getRolePreview,
  updateAdminRole,
  type GoPermission,
  type GoRole,
  type PermissionTreeNode,
  type RolePermissionPreview
} from "@/api/admin";
import { useUserStoreHook } from "@/store/modules/user";
import RePagination from "@/components/RePagination";

defineOptions({ name: "GoAdminRoles" });

const { t } = useI18n();
const loading = ref(false);
const saving = ref(false);
const dialogVisible = ref(false);
const roles = ref<GoRole[]>([]);
const permissions = ref<GoPermission[]>([]);
const permissionTree = ref<PermissionTreeNode[]>([]);
const preview = ref<RolePermissionPreview>();
const selectedRole = ref<GoRole>();
const pagination = reactive({ page: 1, pageSize: 10, total: 0 });
const formRef = ref<FormInstance>();
const permissionTreeRef = ref<InstanceType<typeof ElTree>>();
const userStore = useUserStoreHook();

const form = reactive({
  id: 0,
  name: "",
  description: "",
  permissions: [] as string[]
});

const rules = computed<FormRules>(() => ({
  name: [
    { required: true, message: t("admin.roleNameRequired"), trigger: "blur" },
    { min: 2, max: 50, message: t("admin.roleNameLength"), trigger: "blur" }
  ]
}));

const canWrite = computed(() => userStore.permissions.includes("roles:write"));
const isEditing = computed(() => form.id > 0);
const permissionCount = computed(() => form.permissions.length);

const treeProps = {
  label: "label",
  children: "children"
};

const formatDate = (value: string) => {
  if (!value) return "-";
  return new Date(value).toLocaleString();
};

const loadData = async () => {
  loading.value = true;
  try {
    const [roleRes, permissionRes, treeRes] = await Promise.all([
      getAdminRoles({ page: pagination.page, page_size: pagination.pageSize }),
      getAdminPermissions(),
      getPermissionTree()
    ]);
    roles.value = roleRes.roles ?? [];
    pagination.total = roleRes.total ?? 0;
    permissions.value = permissionRes.permissions ?? [];
    permissionTree.value = treeRes.tree ?? [];
    if (!selectedRole.value && roles.value.length) {
      await selectRole(roles.value[0]);
    }
  } catch {
    message(t("admin.roleLoadFailed"), { type: "error" });
  } finally {
    loading.value = false;
  }
};

const resetForm = () => {
  form.id = 0;
  form.name = "";
  form.description = "";
  form.permissions = [];
  formRef.value?.clearValidate();
  permissionTreeRef.value?.setCheckedKeys([]);
};

const syncTreeChecked = async () => {
  await nextTick();
  permissionTreeRef.value?.setCheckedKeys(form.permissions, false);
};

const openCreate = async () => {
  resetForm();
  dialogVisible.value = true;
  await syncTreeChecked();
};

const openEdit = async (row: GoRole) => {
  form.id = row.id;
  form.name = row.name;
  form.description = row.description;
  form.permissions = [...(row.permissions ?? [])];
  dialogVisible.value = true;
  await syncTreeChecked();
};

const collectCheckedPermissions = () => {
  const checked = permissionTreeRef.value?.getCheckedKeys(false) ?? [];
  const half = permissionTreeRef.value?.getHalfCheckedKeys() ?? [];
  const all = [...checked, ...half].map(String);
  const validCodes = new Set(permissions.value.map(item => item.code));
  form.permissions = Array.from(new Set(all.filter(code => validCodes.has(code))));
};

const submit = async () => {
  collectCheckedPermissions();
  const valid = await formRef.value?.validate().catch(() => false);
  if (!valid) return;

  saving.value = true;
  const payload = {
    name: form.name,
    description: form.description,
    permissions: form.permissions
  };
  try {
    if (isEditing.value) {
      await updateAdminRole(form.id, payload);
      message(t("admin.roleUpdated"), { type: "success" });
    } else {
      await createAdminRole(payload);
      message(t("admin.roleCreated"), { type: "success" });
      pagination.page = 1;
    }
    dialogVisible.value = false;
    await loadData();
  } catch {
    message(isEditing.value ? t("admin.roleUpdateFailed") : t("admin.roleCreateFailed"), { type: "error" });
  } finally {
    saving.value = false;
  }
};

const removeRole = async (row: GoRole) => {
  try {
    await ElMessageBox.confirm(
      t("admin.roleDeleteConfirm", { name: row.name }),
      t("common.tip"),
      {
        type: "warning",
        confirmButtonText: t("common.confirm"),
        cancelButtonText: t("common.cancel")
      }
    );
    await deleteAdminRole(row.id);
    message(t("admin.roleDeleted"), { type: "success" });
    if (roles.value.length === 1 && pagination.page > 1) pagination.page -= 1;
    await loadData();
  } catch (error) {
    if (error !== "cancel") message(t("admin.roleDeleteFailed"), { type: "error" });
  }
};

const selectRole = async (row: GoRole) => {
  selectedRole.value = row;
  try {
    const res = await getRolePreview(row.id);
    preview.value = res.preview;
  } catch {
    preview.value = undefined;
    message(t("admin.roleLoadFailed"), { type: "error" });
  }
};

onMounted(loadData);
</script>

<template>
  <div class="page-container">
    <!-- 头部工具栏 -->
    <div class="page-header">
      <div class="page-header-left">
        <h2 class="page-title">{{ t("routes.roles") }}</h2>
        <span class="page-badge">/api/v1/admin/roles</span>
      </div>
      <el-space>
        <el-button :loading="loading" @click="loadData">
          {{ t("common.refresh") }}
        </el-button>
        <el-button v-if="canWrite" type="primary" @click="openCreate">
          + {{ t("admin.createRole") }}
        </el-button>
      </el-space>
    </div>

    <section class="role-layout">
      <div class="role-panel role-list-panel">
        <div class="panel-head">
          <span>{{ t("routes.roles") }}</span>
          <small>{{ pagination.total }} {{ t("admin.role") }}</small>
        </div>
        <el-table
          v-loading="loading"
          :data="roles"
          row-key="id"
          highlight-current-row
          class="role-table"
          @row-click="selectRole"
        >
          <el-table-column prop="name" :label="t('admin.role')" min-width="150">
            <template #default="{ row }">
              <div class="role-name-cell">
                <span class="role-mark">{{ row.name?.slice(0, 1)?.toUpperCase() }}</span>
                <div>
                  <div class="role-name">{{ row.name }}</div>
                  <div class="role-desc">{{ row.description || t("admin.noDescription") }}</div>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column :label="t('admin.permissions')" width="86" align="center">
            <template #default="{ row }">{{ row.permissions?.length ?? 0 }}</template>
          </el-table-column>
          <el-table-column v-if="canWrite" :label="t('common.operation')" width="130" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click.stop="openEdit(row)">{{ t("common.edit") }}</el-button>
              <el-button
                link
                type="danger"
                :disabled="['admin', 'user'].includes(row.name)"
                @click.stop="removeRole(row)"
              >
                {{ t("common.delete") }}
              </el-button>
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

      <div class="role-panel preview-panel">
        <div class="panel-head">
          <span>{{ t("admin.rolePreviewTitle") }}</span>
          <small>{{ selectedRole?.name || t("admin.notSelected") }}</small>
        </div>
        <div v-if="preview" class="preview-grid">
          <div class="preview-card">
            <div class="preview-title">{{ t("admin.visibleMenus") }}</div>
            <div class="preview-list">
              <el-tag v-for="item in preview.menus" :key="item.id" effect="plain">
                {{ item.label }}
              </el-tag>
              <span v-if="!preview.menus.length" class="empty-text">{{ t("admin.noMenuPermissions") }}</span>
            </div>
          </div>
          <div class="preview-card">
            <div class="preview-title">{{ t("admin.buttonPermissions") }}</div>
            <div class="preview-list">
              <el-tag v-for="item in preview.buttons" :key="item.id" type="warning" effect="plain">
                {{ item.label }}
              </el-tag>
              <span v-if="!preview.buttons.length" class="empty-text">{{ t("admin.noButtonPermissions") }}</span>
            </div>
          </div>
          <div class="preview-card full">
            <div class="preview-title">{{ t("admin.permissionCodes") }}</div>
            <div class="code-list">
              <code v-for="code in preview.permissions" :key="code">{{ code }}</code>
            </div>
          </div>
        </div>
        <el-empty v-else :description="t('admin.selectRolePreview')" />
      </div>
    </section>

    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? t('admin.editRole') : t('admin.createRole')"
      width="760px"
      class="role-dialog"
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="86px">
        <el-form-item :label="t('admin.roleName')" prop="name">
          <el-input v-model="form.name" :placeholder="t('admin.roleNameRequired')" />
        </el-form-item>
        <el-form-item :label="t('common.description')">
          <el-input v-model="form.description" type="textarea" :rows="3" :placeholder="t('admin.descriptionPlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('admin.permissionConfig')">
          <div class="tree-shell">
            <div class="tree-toolbar">
              <span>{{ t("admin.selectedCount", { count: permissionCount }) }}</span>
              <el-button link type="primary" @click="permissionTreeRef?.setCheckedKeys([])">
                {{ t("admin.clear") }}
              </el-button>
            </div>
            <el-tree
              ref="permissionTreeRef"
              :data="permissionTree"
              :props="treeProps"
              node-key="id"
              show-checkbox
              default-expand-all
              check-strictly
              class="permission-tree"
              @check="collectCheckedPermissions"
            >
              <template #default="{ data }">
                <div class="tree-node">
                  <span>{{ data.label }}</span>
                  <code v-if="data.code">{{ data.code }}</code>
                </div>
              </template>
            </el-tree>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="saving" @click="submit">{{ t("common.save") }}</el-button>
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

.role-layout {
  display: grid;
  grid-template-columns: minmax(520px, 1.2fr) minmax(360px, 0.8fr);
  gap: 16px;
}

@media (max-width: 1100px) {
  .role-layout {
    grid-template-columns: 1fr;
  }
}

.role-panel {
  background: var(--app-surface);
  border-radius: 8px;
  border: 1px solid var(--app-border-soft);
  overflow: hidden;
  box-shadow: 0 10px 28px rgb(33 49 77 / 8%);
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 18px;
  font-weight: 700;
  color: var(--app-text);
  border-bottom: 1px solid var(--app-border-soft);
}

.panel-head small {
  color: var(--app-text-muted);
  font-weight: 500;
}

.role-table {
  width: 100%;

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
  }

  :deep(.el-table__cell) {
    border-color: var(--app-border-soft);
  }
}

.role-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.role-mark {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  color: var(--app-primary);
  font-weight: 800;
  background: linear-gradient(135deg, var(--app-bg-soft) 0%, #e8fff7 100%);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.role-name {
  font-weight: 700;
  color: var(--app-text);
}

.role-desc,
.empty-text {
  color: var(--app-text-muted);
  font-size: 12px;
}

.preview-panel {
  min-height: 420px;
}

.preview-grid {
  padding: 16px;
  display: grid;
  gap: 12px;
}

.preview-card {
  padding: 14px;
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 6px;
}

.preview-card.full {
  grid-column: 1 / -1;
}

.preview-title {
  margin-bottom: 10px;
  color: var(--app-text);
  font-size: 14px;
  font-weight: 750;
}

.preview-list,
.code-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.code-list code {
  padding: 4px 7px;
  color: var(--app-primary-strong);
  font-size: 12px;
  background: var(--app-surface-soft);
  border-radius: 6px;
}

.tree-shell {
  width: 100%;
  overflow: hidden;
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.tree-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  color: var(--app-text-secondary);
  background: var(--app-surface-muted);
  border-bottom: 1px solid var(--app-border-soft);
}

.permission-tree {
  padding: 10px 8px 14px;
  max-height: 420px;
  overflow-y: auto;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tree-node code {
  color: var(--app-text-secondary);
  font-size: 11px;
}
</style>
