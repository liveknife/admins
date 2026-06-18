<script setup lang="ts">
import { hasAuth, getAuths } from "@/router/utils";
import { useI18n } from "@/i18n";

defineOptions({
  name: "PermissionButtonRouter"
});

const { t } = useI18n();
const codeText = (code: string) => t("permission.hasCode", { code });
</script>

<template>
  <div>
    <p class="mb-2!">{{ t("permission.authList") }}{{ getAuths() }}</p>

    <el-card shadow="never" class="mb-2">
      <template #header>
        <div class="card-header">{{ t("permission.componentMode") }}</div>
      </template>
      <el-space wrap>
        <Auth value="permission:btn:add">
          <el-button plain type="warning">
            {{ codeText("'permission:btn:add'") }}
          </el-button>
        </Auth>
        <Auth :value="['permission:btn:edit']">
          <el-button plain type="primary">
            {{ codeText("['permission:btn:edit']") }}
          </el-button>
        </Auth>
        <Auth
          :value="[
            'permission:btn:add',
            'permission:btn:edit',
            'permission:btn:delete'
          ]"
        >
          <el-button plain type="danger">
            {{ codeText("['permission:btn:add', 'permission:btn:edit', 'permission:btn:delete']") }}
          </el-button>
        </Auth>
      </el-space>
    </el-card>

    <el-card shadow="never" class="mb-2">
      <template #header>
        <div class="card-header">{{ t("permission.functionMode") }}</div>
      </template>
      <el-space wrap>
        <el-button v-if="hasAuth('permission:btn:add')" plain type="warning">
          {{ codeText("'permission:btn:add'") }}
        </el-button>
        <el-button v-if="hasAuth(['permission:btn:edit'])" plain type="primary">
          {{ codeText("['permission:btn:edit']") }}
        </el-button>
        <el-button
          v-if="
            hasAuth([
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
        <el-button v-auth="'permission:btn:add'" plain type="warning">
          {{ codeText("'permission:btn:add'") }}
        </el-button>
        <el-button v-auth="['permission:btn:edit']" plain type="primary">
          {{ codeText("['permission:btn:edit']") }}
        </el-button>
        <el-button
          v-auth="[
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
