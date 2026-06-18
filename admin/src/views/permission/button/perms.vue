<script setup lang="ts">
import { hasPerms } from "@/utils/auth";
import { useUserStoreHook } from "@/store/modules/user";
import { useI18n } from "@/i18n";

const { permissions } = useUserStoreHook();
const { t } = useI18n();
const codeText = (code: string) => t("permission.hasCode", { code });

defineOptions({
  name: "PermissionButtonLogin"
});
</script>

<template>
  <div>
    <p class="mb-2!">{{ t("permission.authList") }}{{ permissions }}</p>
    <p v-show="permissions?.[0] === '*:*:*'" class="mb-2!">
      {{ t("permission.superAdminTip") }}
    </p>

    <el-card shadow="never" class="mb-2">
      <template #header>
        <div class="card-header">{{ t("permission.componentMode") }}</div>
      </template>
      <el-space wrap>
        <Perms value="permission:btn:add">
          <el-button plain type="warning">
            {{ codeText("'permission:btn:add'") }}
          </el-button>
        </Perms>
        <Perms :value="['permission:btn:edit']">
          <el-button plain type="primary">
            {{ codeText("['permission:btn:edit']") }}
          </el-button>
        </Perms>
        <Perms
          :value="[
            'permission:btn:add',
            'permission:btn:edit',
            'permission:btn:delete'
          ]"
        >
          <el-button plain type="danger">
            {{ codeText("['permission:btn:add', 'permission:btn:edit', 'permission:btn:delete']") }}
          </el-button>
        </Perms>
      </el-space>
    </el-card>

    <el-card shadow="never" class="mb-2">
      <template #header>
        <div class="card-header">{{ t("permission.functionMode") }}</div>
      </template>
      <el-space wrap>
        <el-button v-if="hasPerms('permission:btn:add')" plain type="warning">
          {{ codeText("'permission:btn:add'") }}
        </el-button>
        <el-button
          v-if="hasPerms(['permission:btn:edit'])"
          plain
          type="primary"
        >
          {{ codeText("['permission:btn:edit']") }}
        </el-button>
        <el-button
          v-if="
            hasPerms([
              'permission:btn:add',
              'permission:btn:edit',
              'permission:btn:delete'
            ])
          "
          plain
          type="danger"
        >
          {{ codeText("['permission:btn:add', 'permission:btn:edit', 'permission:btn:delete']") }}
        </el-button>
      </el-space>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          {{ t("permission.directiveMode") }}
        </div>
      </template>
      <el-space wrap>
        <el-button v-perms="'permission:btn:add'" plain type="warning">
          {{ codeText("'permission:btn:add'") }}
        </el-button>
        <el-button v-perms="['permission:btn:edit']" plain type="primary">
          {{ codeText("['permission:btn:edit']") }}
        </el-button>
        <el-button
          v-perms="[
            'permission:btn:add',
            'permission:btn:edit',
            'permission:btn:delete'
          ]"
          plain
          type="danger"
        >
          {{ codeText("['permission:btn:add', 'permission:btn:edit', 'permission:btn:delete']") }}
        </el-button>
      </el-space>
    </el-card>
  </div>
</template>
