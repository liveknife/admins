<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox, type FormInstance, type FormRules } from "element-plus";
import { message } from "@/utils/message";
import { useI18n } from "@/i18n";
import {
  createAdminUser,
  deactivateAdminUser,
  getAdminRoles,
  getAdminUserPassword,
  getAdminUsers,
  resetAdminUserPassword,
  setAdminUserRoles,
  updateAdminUser,
  type GoRole
} from "@/api/admin";
import { encryptPassword } from "@/utils/passwordCrypto";
import type { GoUser } from "@/api/user";
import { useUserStoreHook } from "@/store/modules/user";
import RePagination from "@/components/RePagination";

const loading = ref(false);
const saving = ref(false);
const users = ref<GoUser[]>([]);
const roles = ref<GoRole[]>([]);
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
});
// 角色弹窗
const roleDialogVisible = ref(false);
// 密码弹窗（重置）
const passwordDialogVisible = ref(false);
// 查看密码弹窗
const viewPasswordDialogVisible = ref(false);
// 新增/编辑用户弹窗
const userFormDialogVisible = ref(false);
const isEditMode = ref(false);

const currentUser = ref<GoUser>();
const selectedRoles = ref<string[]>([]);
const visiblePassword = ref("");
const passwordFormRef = ref<FormInstance>();
const userFormRef = ref<FormInstance>();
const userStore = useUserStoreHook();
const { t } = useI18n();

// ── 密码重置表单 ──
const passwordForm = reactive({
  password: ""
});

const passwordRules = computed<FormRules>(() => ({
  password: [
    { required: true, message: t("admin.newPasswordRequired"), trigger: "blur" },
    { min: 6, max: 72, message: t("admin.passwordLength"), trigger: "blur" }
  ]
}));

// ── 用户表单（新增/编辑共用）──
const userForm = reactive({
  username: "",
  email: "",
  phone: "",
  password: "",
  roles: [] as string[]
});

const userFormRules = computed<FormRules>(() => {
  const rules: FormRules = {
    username: [
      { required: true, message: t("admin.usernameRequired"), trigger: "blur" },
      { min: 2, max: 50, message: "2-50 characters", trigger: "blur" }
    ],
    email: [
      { required: true, message: t("admin.emailRequired"), trigger: "blur" },
      { type: "email", message: "invalid email", trigger: "blur" }
    ]
  };
  // 编辑模式不需要密码
  if (!isEditMode.value) {
    rules.password = [
      { required: true, message: t("admin.passwordRequired"), trigger: "blur" },
      { min: 6, max: 72, message: t("admin.passwordLength"), trigger: "blur" }
    ];
  }
  return rules;
});

const resetUserForm = () => {
  userForm.username = "";
  userForm.email = "";
  userForm.phone = "";
  userForm.password = "";
  userForm.roles = [];
  isEditMode.value = false;
  currentUser.value = undefined;
};

const canWrite = computed(() => userStore.permissions.includes("users:write"));
const canReadPassword = computed(() =>
  userStore.permissions.includes("users:password:read")
);

const isDeactivated = (row: GoUser) => Boolean(row.deleted_at);

const formatDate = (value: string) => {
  if (!value) return "-";
  return new Date(value).toLocaleString();
};

const loadData = async () => {
  loading.value = true;
  try {
    const [userRes, roleRes] = await Promise.all([
      getAdminUsers({
        page: pagination.page,
        page_size: pagination.pageSize
      }),
      getAdminRoles({ page: 1, page_size: 100 })
    ]);
    users.value = userRes.users ?? [];
    pagination.total = userRes.total ?? 0;
    roles.value = roleRes.roles ?? [];
  } catch (error) {
    message(t("admin.userListLoadFailed"), { type: "error" });
  } finally {
    loading.value = false;
  }
};

// ── 打开新增弹窗 ──
const openCreateDialog = () => {
  resetUserForm();
  isEditMode.value = false;
  userFormRef.value?.clearValidate();
  userFormDialogVisible.value = true;
};

// ── 打开编辑弹窗 ──
const openEditDialog = (row: GoUser) => {
  resetUserForm();
  isEditMode.value = true;
  userFormRef.value?.clearValidate();
  currentUser.value = row;
  userForm.username = row.username;
  userForm.email = row.email;
  userForm.phone = row.phone ?? "";
  userForm.password = ""; // 编辑时默认为空，不修改密码
  userForm.roles = [...(row.roles ?? [])];
  userFormDialogVisible.value = true;
};

// ── 提交用户新增/编辑 ──
const submitUserForm = async () => {
  const valid = await userFormRef.value?.validate().catch(() => false);
  if (!valid) return;

  saving.value = true;
  try {
    if (isEditMode.value && currentUser.value) {
      // 编辑模式：只传基本信息（不包含密码）
      const payload = {
        username: userForm.username,
        email: userForm.email,
        phone: userForm.phone || undefined
      };
      await updateAdminUser(currentUser.value.id, payload);
      // 如果同时修改了角色
      if (userForm.roles.length > 0) {
        const currentRoles = currentUser.value.roles ?? [];
        if (
          userForm.roles.length !== currentRoles.length ||
          !userForm.roles.every(r => currentRoles.includes(r))
        ) {
          await setAdminUserRoles(currentUser.value.id, userForm.roles);
        }
      }
      message(t("admin.userUpdated"), { type: "success" });
    } else {
      // 新增模式：需要加密密码
      const passwordEncrypted = await encryptPassword(userForm.password);
      await createAdminUser({
        username: userForm.username,
        email: userForm.email,
        phone: userForm.phone || undefined,
        password_encrypted: passwordEncrypted,
        roles: userForm.roles.length > 0 ? userForm.roles : undefined
      });
      message(t("admin.userCreated"), { type: "success" });
    }

    userFormDialogVisible.value = false;
    if (!isEditMode.value) pagination.page = 1;
    await loadData();
  } catch (error: any) {
    const msg = error?.response?.data?.error || (isEditMode.value ? t("admin.userUpdateFailed") : t("admin.userCreateFailed"));
    message(msg, { type: "error" });
  } finally {
    saving.value = false;
  }
};

// ── 角色 ──
const openRoleDialog = (row: GoUser) => {
  currentUser.value = row;
  selectedRoles.value = [...(row.roles ?? [])];
  roleDialogVisible.value = true;
};

const submitRoles = async () => {
  if (!currentUser.value) return;
  saving.value = true;
  try {
    await setAdminUserRoles(currentUser.value.id, selectedRoles.value);
    message(t("admin.userRoleUpdated"), { type: "success" });
    roleDialogVisible.value = false;
    await loadData();
  } catch (error) {
    message(t("admin.userRoleUpdateFailed"), { type: "error" });
  } finally {
    saving.value = false;
  }
};

// ── 重置密码 ──
const openPasswordDialog = (row: GoUser) => {
  currentUser.value = row;
  passwordForm.password = "";
  passwordDialogVisible.value = true;
};

const submitPassword = async () => {
  const valid = await passwordFormRef.value?.validate().catch(() => false);
  if (!valid || !currentUser.value) return;

  saving.value = true;
  try {
    await resetAdminUserPassword(currentUser.value.id, passwordForm.password);
    message(t("admin.passwordReset"), { type: "success" });
    passwordDialogVisible.value = false;
  } catch (error) {
    message(t("admin.passwordResetFailed"), { type: "error" });
  } finally {
    saving.value = false;
  }
};

// ── 查看密码 ──
const viewPassword = async (row: GoUser) => {
  try {
    await ElMessageBox.confirm(
      t("admin.viewPasswordConfirm", { name: row.username }),
      t("admin.sensitiveAction"),
      { type: "warning" }
    );
    const res = await getAdminUserPassword(row.id);
    currentUser.value = row;
    visiblePassword.value = res.password;
    viewPasswordDialogVisible.value = true;
  } catch (error) {
    if (error !== "cancel") message(t("admin.viewPasswordFailed"), { type: "error" });
  }
};

// ── 停用/删除用户 ──
const deactivateUser = async (row: GoUser) => {
  try {
    await ElMessageBox.confirm(
      t("admin.userDeactivateConfirm", { name: row.username }),
      t("admin.deactivate"),
      { type: "warning" }
    );
    await deactivateAdminUser(row.id);
    message(t("admin.userDeactivated"), { type: "success" });
    if (users.value.length === 1 && pagination.page > 1) {
      pagination.page -= 1;
    }
    await loadData();
  } catch (error) {
    if (error !== "cancel") message(t("admin.userDeactivateFailed"), { type: "error" });
  }
};

const handlePageChange = () => {
  loadData();
};

const confirmClose = done => {
  if (saving.value) return;
  ElMessageBox.confirm(t("admin.closeConfirm"), t("common.tip"), {
    type: "warning"
  })
    .then(() => done())
    .catch(() => undefined);
};

onMounted(loadData);
</script>

<template>
  <div class="page-container">
    <!-- 头部工具栏 -->
    <div class="page-header">
      <div class="page-header-left">
        <h2 class="page-title">{{ t("routes.users") }}</h2>
        <span class="page-badge">/api/v1/admin/users</span>
      </div>
      <el-space>
        <el-button v-if="canWrite" type="primary" @click="openCreateDialog">
          + {{ t("admin.createUser") }}
        </el-button>
        <el-button :loading="loading" @click="loadData">
          {{ t("common.refresh") }}
        </el-button>
      </el-space>
    </div>

    <!-- 用户表格 -->
    <div class="table-card">
      <el-table
        v-loading="loading"
        :data="users"
        stripe
        class="admin-table"
        row-key="id"
      >
        <el-table-column prop="id" label="ID" width="76" align="center" />
        <el-table-column prop="username" :label="t('admin.username')" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <span class="user-avatar">{{ row.username?.slice(0, 1)?.toUpperCase() }}</span>
              <span class="user-name">{{ row.username }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="phone" :label="t('admin.phone')" min-width="150">
          <template #default="{ row }">
            <span class="muted-text">{{ row.phone || "-" }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="email" :label="t('admin.email')" min-width="220">
          <template #default="{ row }">
            <span class="email-text">{{ row.email }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('admin.status')" width="90">
          <template #default="{ row }">
            <el-tag
              size="small"
              :type="isDeactivated(row) ? 'info' : 'success'"
              effect="light"
              round
            >
              {{ isDeactivated(row) ? t("admin.deactivated") : t("admin.normal") }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('admin.role')" min-width="180">
          <template #default="{ row }">
            <div class="tag-list">
              <el-tag
                v-for="role in row.roles"
                :key="role"
                size="small"
                type="primary"
                effect="plain"
                round
              >
                {{ role }}
              </el-tag>
              <span v-if="!row.roles?.length" class="muted-text">-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('admin.permissions')" min-width="220">
          <template #default="{ row }">
            <div class="tag-list">
              <el-tag
                v-for="permission in row.permissions?.slice(0, 3)"
                :key="permission"
                size="small"
                type="info"
                effect="plain"
                round
              >
                {{ permission }}
              </el-tag>
              <el-popover
                v-if="row.permissions?.length > 3"
                placement="top"
                width="320"
                trigger="hover"
              >
                <template #reference>
                  <el-tag size="small" type="info" effect="dark" round>
                    +{{ row.permissions.length - 3 }}
                  </el-tag>
                </template>
                <div class="tag-list tag-list--popover">
                  <el-tag
                    v-for="permission in row.permissions"
                    :key="permission"
                    size="small"
                    type="info"
                    effect="plain"
                    round
                  >
                    {{ permission }}
                  </el-tag>
                </div>
              </el-popover>
              <span v-if="!row.permissions?.length" class="muted-text">-</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column :label="t('admin.createdAt')" width="175">
          <template #default="{ row }">
            <span class="muted-text">{{ formatDate(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column
          v-if="canWrite || canReadPassword"
          :label="t('common.operation')"
          width="330"
          fixed="right"
        >
          <template #default="{ row }">
            <div class="action-list">
              <el-button
                v-if="canWrite"
                type="primary"
                link
                :disabled="isDeactivated(row)"
                @click="openEditDialog(row)"
              >
                {{ t("common.edit") }}
              </el-button>
              <el-button
                v-if="canWrite"
                type="primary"
                link
                :disabled="isDeactivated(row)"
                @click="openRoleDialog(row)"
              >
                {{ t("admin.setRole") }}
              </el-button>
              <el-button
                v-if="canWrite"
                type="warning"
                link
                :disabled="isDeactivated(row)"
                @click="openPasswordDialog(row)"
              >
                {{ t("admin.resetPassword") }}
              </el-button>
              <el-button
                v-if="canReadPassword"
                type="danger"
                link
                :disabled="isDeactivated(row)"
                @click="viewPassword(row)"
              >
                {{ t("admin.viewPassword") }}
              </el-button>
              <el-button
                v-if="canWrite"
                type="danger"
                link
                :disabled="isDeactivated(row)"
                @click="deactivateUser(row)"
              >
                {{ t("admin.deactivate") }}
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <RePagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :disabled="loading"
        @change="handlePageChange"
      />
    </div>

    <!-- ═══ 新增/编辑用户弹窗 ═══ -->
    <el-dialog
      v-model="userFormDialogVisible"
      :title="isEditMode ? t('admin.editUser') : t('admin.createUser')"
      width="520px"
      :before-close="confirmClose"
      @closed="resetUserForm()"
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userFormRules"
        label-width="84px"
      >
        <el-form-item :label="t('admin.username')" prop="username">
          <el-input
            v-model="userForm.username"
            :placeholder="t('admin.usernamePlaceholder')"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>
        <el-form-item :label="t('admin.email')" prop="email">
          <el-input
            v-model="userForm.email"
            :placeholder="t('admin.emailPlaceholder')"
            maxlength="255"
          />
        </el-form-item>
        <el-form-item :label="t('admin.phone')" prop="phone">
          <el-input
            v-model="userForm.phone"
            :placeholder="t('admin.phoneOptional')"
            maxlength="20"
          />
        </el-form-item>
        <el-form-item
          v-if="!isEditMode"
          :label="t('admin.password')"
          prop="password"
        >
          <el-input
            v-model="userForm.password"
            show-password
            autocomplete="new-password"
            :placeholder="t('admin.passwordPlaceholder')"
          />
        </el-form-item>
        <el-form-item :label="t('admin.role')">
          <el-select
            v-model="userForm.roles"
            multiple
            clearable
            class="w-full"
            :placeholder="t('admin.rolePlaceholder')"
          >
            <el-option
              v-for="role in roles"
              :key="role.id"
              :label="role.name"
              :value="role.name"
            />
          </el-select>
          <span class="form-hint">{{ t("admin.roleHint") }}</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userFormDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitUserForm">
          {{ isEditMode ? t("common.save") : t("admin.create") }}
        </el-button>
      </template>
    </el-dialog>

    <!-- ═══ 设置角色弹窗 ═══ -->
    <el-dialog
      v-model="roleDialogVisible"
      :title="t('admin.setUserRole')"
      width="440px"
      :before-close="confirmClose"
    >
      <el-form label-width="60px">
        <el-form-item :label="t('admin.user')">
          <span class="dialog-label-value">{{ currentUser?.username }}</span>
        </el-form-item>
        <el-form-item :label="t('admin.role')">
          <el-select
            v-model="selectedRoles"
            multiple
            clearable
            class="w-full"
            :placeholder="t('admin.rolePlaceholder')"
          >
            <el-option
              v-for="role in roles"
              :key="role.id"
              :label="role.name"
              :value="role.name"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitRoles">
          {{ t("common.save") }}
        </el-button>
      </template>
    </el-dialog>

    <!-- ═══ 重置密码弹窗 ═══ -->
    <el-dialog
      v-model="passwordDialogVisible"
      :title="t('admin.resetPassword')"
      width="440px"
      :before-close="confirmClose"
      @closed="passwordForm.password = ''"
    >
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordRules"
        label-width="84px"
      >
        <el-form-item :label="t('admin.user')">
          <span class="dialog-label-value">{{ currentUser?.username }}</span>
        </el-form-item>
        <el-form-item :label="t('admin.newPassword')" prop="password">
          <el-input
            v-model="passwordForm.password"
            show-password
            autocomplete="new-password"
            :placeholder="t('admin.newPasswordRequired')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="passwordDialogVisible = false">{{ t("common.cancel") }}</el-button>
        <el-button type="primary" :loading="saving" @click="submitPassword">
          {{ t("common.save") }}
        </el-button>
      </template>
    </el-dialog>

    <!-- ═══ 查看密码弹窗 ═══ -->
    <el-dialog v-model="viewPasswordDialogVisible" :title="t('admin.viewPassword')" width="440px">
      <el-alert
        class="mb-4!"
        type="warning"
        show-icon
        :closable="false"
        :title="t('admin.passwordReadNotice')"
      />
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="t('admin.user')">
          {{ currentUser?.username }}
        </el-descriptions-item>
        <el-descriptions-item :label="t('admin.password')">
          <el-input v-model="visiblePassword" readonly show-password />
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="viewPasswordDialogVisible = false">{{ t("common.close") }}</el-button>
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

.email-text {
  color: var(--app-text-secondary);
  font-family: "Menlo", "Monaco", monospace;
  font-size: 12.5px;
}

.muted-text {
  color: var(--app-text-muted);
}

.tag-list {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-list--popover {
  max-height: 180px;
  overflow-y: auto;
}

.action-list {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 10px;

  :deep(.el-button) {
    height: 22px;
    margin-left: 0;
    padding: 0;
    font-weight: 600;
  }
}

.form-hint {
  display: block;
  margin-top: 5px;
  font-size: 12px;
  color: var(--app-text-muted);
}

.dialog-label-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--app-text-secondary);
}
</style>
