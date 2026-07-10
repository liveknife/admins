<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  getSystemSettings,
  saveSystemSettings,
  type SystemSetting
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "SystemToolsSettings" });

const loading = ref(false);
const saving = ref(false);
const settings = ref<SystemSetting[]>([]);

const groupOrder = ["site", "rag", "ai", "ops", "general"];

const grouped = computed(() => {
  const groups: Record<string, SystemSetting[]> = {};
  for (const item of settings.value) {
    const key = item.group_name || "general";
    if (!groups[key]) groups[key] = [];
    groups[key].push(item);
  }
  return Object.keys(groups)
    .sort((a, b) => groupOrder.indexOf(a) - groupOrder.indexOf(b))
    .map(key => ({ key, items: groups[key] }));
});

const groupLabel = (value: string) =>
  ({
    site: "官网",
    rag: "知识库",
    ai: "AI",
    ops: "运营",
    general: "通用"
  })[value] ?? value;

const keyLabels: Record<string, string> = {
  "site.name": "官网名称",
  "site.description": "官网描述",
  "site.maintenance": "官网维护模式",
  "rag.default_visibility": "上传文档默认可见性",
  "rag.public_enabled": "允许官网知识库公开问答",
  "ai.log_retention_days": "AI 调用日志保留天数",
  "ops.alert_email": "运营告警接收邮箱"
};

const publicKeys = new Set(Object.keys(keyLabels));

const settingTitle = (item: SystemSetting) =>
  item.description || keyLabels[item.setting_key] || item.setting_key;

const settingHint = (item: SystemSetting) =>
  item.description ? item.setting_key : "未填写配置说明";

const isSensitive = (item: SystemSetting) => {
  if (publicKeys.has(item.setting_key)) return false;
  return (
    item.is_secret ||
    /api[_-]?key|token|secret|password/i.test(item.setting_key)
  );
};

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getSystemSettings();
    settings.value = res.settings ?? [];
  } catch {
    message("系统配置加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const saveData = async () => {
  saving.value = true;
  try {
    const payload = settings.value.map(item => ({
      ...item,
      is_secret: isSensitive(item)
    }));
    const res = await saveSystemSettings(payload);
    settings.value = res.settings ?? [];
    message("系统配置已保存", { type: "success" });
  } catch {
    message("系统配置保存失败", { type: "error" });
  } finally {
    saving.value = false;
  }
};

const saveSingleSetting = async (
  item: SystemSetting,
  value?: string | number | boolean
) => {
  if (value !== undefined) {
    item.setting_value = String(value);
  }
  saving.value = true;
  try {
    const res = await saveSystemSettings([
      {
        ...item,
        is_secret: isSensitive(item)
      }
    ]);
    const next = res.settings?.find(
      setting => setting.setting_key === item.setting_key
    );
    if (next) {
      Object.assign(item, next);
    }
    message("Setting saved", { type: "success" });
  } catch {
    message("Setting save failed", { type: "error" });
    await loadData();
  } finally {
    saving.value = false;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="settings-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Config center</p>
        <h2>系统配置中心</h2>
        <p>集中维护官网、知识库、AI 与运营相关开关，配置项会持久化到数据库。</p>
      </div>
      <div class="head-actions">
        <el-button @click="loadData">刷新</el-button>
        <el-button type="primary" :loading="saving" @click="saveData">保存配置</el-button>
      </div>
    </section>

    <section class="settings-grid">
      <div v-for="group in grouped" :key="group.key" class="panel">
        <div class="panel-title">
          <h3>{{ groupLabel(group.key) }}</h3>
          <span>{{ group.items.length }} 项</span>
        </div>
        <div class="setting-list">
          <div
            v-for="item in group.items"
            :key="item.setting_key"
            class="setting-row"
          >
            <div class="setting-meta">
              <strong>{{ settingTitle(item) }}</strong>
              <span>{{ settingHint(item) }}</span>
            </div>

            <el-switch
              v-if="item.value_type === 'boolean'"
              v-model="item.setting_value"
              active-value="true"
              inactive-value="false"
              :loading="saving"
              @change="value => saveSingleSetting(item, value)"
            />
            <el-input
              v-else-if="item.value_type === 'number'"
              v-model="item.setting_value"
              type="number"
              min="0"
              class="setting-control"
            />
            <el-select
              v-else-if="item.value_type === 'select'"
              v-model="item.setting_value"
              class="setting-control"
            >
              <el-option label="internal" value="internal" />
              <el-option label="public" value="public" />
            </el-select>
            <el-input
              v-else-if="item.value_type === 'textarea'"
              v-model="item.setting_value"
              type="textarea"
              :rows="2"
              class="setting-control"
            />
            <el-input
              v-else-if="isSensitive(item)"
              v-model="item.setting_value"
              type="password"
              show-password
              class="setting-control"
            />
            <el-input
              v-else
              v-model="item.setting_value"
              class="setting-control"
            />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.settings-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.page-head,
.panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.page-head {
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
  font-weight: 800;
  text-transform: uppercase;
}

.page-head h2,
.panel-title h3 {
  margin: 0;
}

.page-head p:last-child {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.head-actions {
  display: flex;
  gap: 8px;
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.panel {
  padding: 18px;
}

.panel-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  color: var(--app-text-secondary);
}

.setting-list {
  display: grid;
}

.setting-row {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(240px, 360px);
  gap: 16px;
  align-items: center;
  min-height: 76px;
  padding: 12px 0;
  border-top: 1px solid var(--app-border);
}

.setting-meta {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.setting-meta strong {
  color: var(--app-text);
  font-size: 15px;
  font-weight: 760;
}

.setting-meta span {
  overflow: hidden;
  color: var(--app-text-secondary);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.setting-control {
  width: 100%;
}

@media (max-width: 980px) {
  .settings-grid,
  .setting-row,
  .page-head {
    grid-template-columns: 1fr;
    display: grid;
  }
}
</style>
