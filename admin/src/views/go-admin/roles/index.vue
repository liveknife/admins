<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from "vue";
import type { FormInstance, FormRules } from "element-plus";
import { ElMessageBox, type ElTree } from "element-plus";
import { message } from "@/utils/message";
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
    { required: true, message: "请输入角色名称", trigger: "blur" },
    { min: 2, max: 50, message: "角色名称长度为 2-50 个字符", trigger: "blur" }
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
    message("角色权限加载失败", { type: "error" });
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
      message("角色已更新", { type: "success" });
    } else {
      await createAdminRole(payload);
      message("角色已创建", { type: "success" });
      pagination.page = 1;
    }
    dialogVisible.value = false;
    await loadData();
  } catch {
    message(isEditing.value ? "角色更新失败" : "角色创建失败", { type: "error" });
  } finally {
    saving.value = false;
  }
};

const removeRole = async (row: GoRole) => {
  try {
    await ElMessageBox.confirm(`确认删除角色 ${row.name}？`, "提示", { type: "warning" });
    await deleteAdminRole(row.id);
    message("角色已删除", { type: "success" });
    if (roles.value.length === 1 && pagination.page > 1) pagination.page -= 1;
    await loadData();
  } catch (error) {
    if (error !== "cancel") message("角色删除失败", { type: "error" });
  }
};

const selectRole = async (row: GoRole) => {
  selectedRole.value = row;
  try {
    const res = await getRolePreview(row.id);
    preview.value = res.preview;
  } catch {
    preview.value = undefined;
    message("角色预览加载失败", { type: "error" });
  }
};

onMounted(loadData);
</script>

<template>
  <div class="role-console">
    <section class="role-hero">
      <div>
        <p class="eyebrow">Access control console</p>
        <h2>权限可视化配置</h2>
        <p>用树形结构维护菜单、按钮和系统能力，实时预览角色能看到的后台入口。</p>
      </div>
      <div class="hero-actions">
        <el-button :loading="loading" @click="loadData">刷新</el-button>
        <el-button v-if="canWrite" type="primary" @click="openCreate">新建角色</el-button>
      </div>
    </section>

    <section class="role-layout">
      <div class="role-panel role-list-panel">
        <div class="panel-head">
          <span>角色列表</span>
          <small>{{ pagination.total }} 个角色</small>
        </div>
        <el-table
          v-loading="loading"
          :data="roles"
          row-key="id"
          highlight-current-row
          class="role-table"
          @row-click="selectRole"
        >
          <el-table-column prop="name" label="角色" min-width="150">
            <template #default="{ row }">
              <div class="role-name-cell">
                <span class="role-mark">{{ row.name?.slice(0, 1)?.toUpperCase() }}</span>
                <div>
                  <div class="role-name">{{ row.name }}</div>
                  <div class="role-desc">{{ row.description || "暂无描述" }}</div>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="权限" width="86" align="center">
            <template #default="{ row }">{{ row.permissions?.length ?? 0 }}</template>
          </el-table-column>
          <el-table-column v-if="canWrite" label="操作" width="130" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click.stop="openEdit(row)">编辑</el-button>
              <el-button
                link
                type="danger"
                :disabled="['admin', 'user'].includes(row.name)"
                @click.stop="removeRole(row)"
              >
                删除
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
          <span>角色权限预览</span>
          <small>{{ selectedRole?.name || "未选择" }}</small>
        </div>
        <div v-if="preview" class="preview-grid">
          <div class="preview-card">
            <div class="preview-title">可见菜单</div>
            <div class="preview-list">
              <el-tag v-for="item in preview.menus" :key="item.id" effect="plain">
                {{ item.label }}
              </el-tag>
              <span v-if="!preview.menus.length" class="empty-text">暂无菜单权限</span>
            </div>
          </div>
          <div class="preview-card">
            <div class="preview-title">按钮权限</div>
            <div class="preview-list">
              <el-tag v-for="item in preview.buttons" :key="item.id" type="warning" effect="plain">
                {{ item.label }}
              </el-tag>
              <span v-if="!preview.buttons.length" class="empty-text">暂无按钮权限</span>
            </div>
          </div>
          <div class="preview-card full">
            <div class="preview-title">权限码</div>
            <div class="code-list">
              <code v-for="code in preview.permissions" :key="code">{{ code }}</code>
            </div>
          </div>
        </div>
        <el-empty v-else description="选择左侧角色查看预览" />
      </div>
    </section>

    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑角色权限' : '新建角色'"
      width="760px"
      class="role-dialog"
      @closed="resetForm"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="86px">
        <el-form-item label="角色名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入角色名称" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="说明这个角色适合谁使用" />
        </el-form-item>
        <el-form-item label="权限配置">
          <div class="tree-shell">
            <div class="tree-toolbar">
              <span>已选择 {{ permissionCount }} 项权限</span>
              <el-button link type="primary" @click="permissionTreeRef?.setCheckedKeys([])">清空</el-button>
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
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存权限</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.role-console {
  padding: 24px;
  display: grid;
  gap: 16px;
}

.role-hero,
.role-panel,
.preview-card,
.tree-shell {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.role-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 22px 24px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-primary);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
}

.role-hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 750;
}

.role-hero p:last-child {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.hero-actions {
  display: flex;
  gap: 10px;
}

.role-layout {
  display: grid;
  grid-template-columns: minmax(520px, 1.2fr) minmax(360px, 0.8fr);
  gap: 16px;
}

.role-panel {
  overflow: hidden;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 18px;
  font-weight: 700;
  border-bottom: 1px solid var(--app-border-soft);
}

.panel-head small {
  color: var(--app-text-muted);
  font-weight: 500;
}

.role-table {
  width: 100%;
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
  background: var(--app-bg-soft);
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

@media (max-width: 1100px) {
  .role-layout {
    grid-template-columns: 1fr;
  }
}
</style>
