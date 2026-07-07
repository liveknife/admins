<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  deleteAIModelConfig,
  getAIModelConfigs,
  saveAIModelConfig,
  setDefaultAIModelConfig,
  testAIModelConfig,
  type AIModelConfig
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGAIModels" });

type ProviderPreset = {
  label: string;
  value: string;
  apiFormat: string;
  baseURL: string;
  chatModel: string;
  embeddingModel: string;
};

type AIModelForm = Partial<AIModelConfig> & {
  api_key: string;
};

const providerPresets: ProviderPreset[] = [
  {
    label: "DeepSeek",
    value: "deepseek",
    apiFormat: "openai",
    baseURL: "https://api.deepseek.com/v1",
    chatModel: "deepseek-chat",
    embeddingModel: ""
  },
  {
    label: "OpenAI / GPT",
    value: "openai",
    apiFormat: "openai",
    baseURL: "https://api.openai.com/v1",
    chatModel: "",
    embeddingModel: ""
  },
  {
    label: "Claude",
    value: "anthropic",
    apiFormat: "anthropic",
    baseURL: "https://api.anthropic.com/v1",
    chatModel: "",
    embeddingModel: ""
  },
  {
    label: "Mimo / 兼容接口",
    value: "mimo",
    apiFormat: "openai",
    baseURL: "",
    chatModel: "",
    embeddingModel: ""
  },
  {
    label: "Ollama",
    value: "ollama",
    apiFormat: "openai",
    baseURL: "http://localhost:11434/v1",
    chatModel: "",
    embeddingModel: ""
  },
  {
    label: "自定义",
    value: "custom",
    apiFormat: "custom",
    baseURL: "",
    chatModel: "",
    embeddingModel: ""
  }
];

const emptyForm = (): AIModelForm => ({
  name: "",
  provider: "deepseek",
  api_format: "openai",
  base_url: "https://api.deepseek.com/v1",
  api_key: "",
  chat_model: "deepseek-chat",
  embedding_model: "",
  temperature: 0.2,
  max_tokens: 0,
  timeout_seconds: 45,
  extra_json: "",
  enabled: true,
  is_default: false
});

const loading = ref(false);
const saving = ref(false);
const testingId = ref<number>();
const configs = ref<AIModelConfig[]>([]);
const current = ref<AIModelConfig>();
const form = reactive<AIModelForm>(emptyForm());

const activeConfig = computed(() => configs.value.find(item => item.is_default));
const enabledCount = computed(() => configs.value.filter(item => item.enabled).length);

const providerLabel = (value?: string) =>
  providerPresets.find(item => item.value === value)?.label ?? value ?? "-";

const formatLabel = (value?: string) => {
  if (value === "anthropic") return "Anthropic";
  if (value === "custom") return "Custom";
  return "OpenAI compatible";
};

const resetForm = () => {
  Object.assign(form, emptyForm());
  current.value = undefined;
};

const applyPreset = (provider: string) => {
  const preset = providerPresets.find(item => item.value === provider);
  if (!preset) return;
  form.provider = preset.value;
  form.api_format = preset.apiFormat;
  form.base_url = preset.baseURL;
  if (!current.value) {
    form.name = preset.label;
  }
  if (!form.chat_model) form.chat_model = preset.chatModel;
  if (!form.embedding_model) form.embedding_model = preset.embeddingModel;
};

const editConfig = (item: AIModelConfig) => {
  current.value = item;
  Object.assign(form, {
    ...item,
    api_key: ""
  });
};

const loadConfigs = async () => {
  loading.value = true;
  try {
    const res = await getAIModelConfigs();
    configs.value = res.configs ?? [];
    if (!current.value && configs.value.length) {
      editConfig(configs.value[0]);
    }
  } catch {
    message("大模型配置加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const saveConfig = async () => {
  saving.value = true;
  try {
    const payload = { ...form };
    if (!payload.api_key) delete payload.api_key;
    const res = await saveAIModelConfig(payload, current.value?.id);
    message("大模型配置已保存", { type: "success" });
    await loadConfigs();
    editConfig(res.config);
  } catch {
    message("大模型配置保存失败", { type: "error" });
  } finally {
    saving.value = false;
  }
};

const activateConfig = async (item: AIModelConfig) => {
  try {
    const res = await setDefaultAIModelConfig(item.id);
    message("已切换默认大模型", { type: "success" });
    await loadConfigs();
    editConfig(res.config);
  } catch {
    message("默认大模型切换失败", { type: "error" });
  }
};

const testConfig = async (item: AIModelConfig) => {
  testingId.value = item.id;
  try {
    const res = await testAIModelConfig(item.id);
    const ok = res.config.last_test_status === "success";
    message(ok ? "模型连接测试通过" : res.config.last_test_message || "模型连接测试失败", {
      type: ok ? "success" : "error"
    });
    await loadConfigs();
    editConfig(res.config);
  } catch {
    message("模型连接测试失败", { type: "error" });
  } finally {
    testingId.value = undefined;
  }
};

const removeConfig = async (item: AIModelConfig) => {
  try {
    await deleteAIModelConfig(item.id);
    message("大模型配置已删除", { type: "success" });
    resetForm();
    await loadConfigs();
  } catch {
    message("大模型配置删除失败", { type: "error" });
  }
};

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";

onMounted(loadConfigs);
</script>

<template>
  <div class="ai-models-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Model gateway</p>
        <h2>大模型配置</h2>
        <p>统一维护 DeepSeek、GPT、Claude、Mimo、Ollama 等模型参数，当前默认配置会被 RAG 与 AI 助手优先使用。</p>
      </div>
      <div class="head-stats">
        <div>
          <span>默认模型</span>
          <strong>{{ activeConfig?.name ?? "-" }}</strong>
        </div>
        <div>
          <span>启用配置</span>
          <strong>{{ enabledCount }}</strong>
        </div>
        <el-button type="primary" @click="resetForm">
          <IconifyIconOnline icon="ri:add-line" />
          新增配置
        </el-button>
      </div>
    </section>

    <section class="model-layout">
      <div class="config-list">
        <div
          v-for="item in configs"
          :key="item.id"
          class="config-item"
          :class="{ active: current?.id === item.id }"
          @click="editConfig(item)"
        >
          <div class="config-main">
            <strong>{{ item.name }}</strong>
            <span>{{ providerLabel(item.provider) }} · {{ formatLabel(item.api_format) }}</span>
          </div>
          <div class="config-badges">
            <el-tag v-if="item.is_default" type="success" effect="light">默认</el-tag>
            <el-tag v-if="!item.enabled" type="info" effect="plain">停用</el-tag>
            <el-tag v-if="item.last_test_status" :type="item.last_test_status === 'success' ? 'success' : 'danger'" effect="light">
              {{ item.last_test_status === "success" ? "测试通过" : "测试失败" }}
            </el-tag>
          </div>
        </div>
        <el-empty v-if="!configs.length" description="暂无模型配置" />
      </div>

      <div class="config-editor">
        <div class="editor-title">
          <div>
            <h3>{{ current ? "编辑配置" : "新增配置" }}</h3>
            <p v-if="current">最近更新：{{ timeText(current.updated_at) }}</p>
          </div>
          <div v-if="current" class="editor-actions">
            <el-button :disabled="current.is_default" @click="activateConfig(current)">设为默认</el-button>
            <el-button :loading="testingId === current.id" @click="testConfig(current)">测试连接</el-button>
            <el-popconfirm title="确认删除这个模型配置？" @confirm="removeConfig(current)">
              <template #reference>
                <el-button type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>

        <el-form :model="form" label-width="118px" class="model-form" @submit.prevent>
          <el-form-item label="配置名称" required>
            <el-input v-model="form.name" placeholder="例如 DeepSeek 生产环境" />
          </el-form-item>

          <div class="form-grid two-columns">
            <el-form-item label="模型厂商">
              <el-select v-model="form.provider" @change="applyPreset">
                <el-option
                  v-for="item in providerPresets"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="接口格式">
              <el-select v-model="form.api_format">
                <el-option label="OpenAI compatible" value="openai" />
                <el-option label="Anthropic" value="anthropic" />
                <el-option label="Custom" value="custom" />
              </el-select>
            </el-form-item>
          </div>

          <el-form-item label="Base URL">
            <el-input v-model="form.base_url" placeholder="https://api.example.com/v1" />
          </el-form-item>

          <el-form-item label="API Key">
            <el-input
              v-model="form.api_key"
              type="password"
              show-password
              autocomplete="new-password"
              :placeholder="current?.has_api_key ? `已保存：${current.masked_api_key}` : '请输入 API Key'"
            />
          </el-form-item>

          <div class="form-grid two-columns">
            <el-form-item label="Chat Model">
              <el-input v-model="form.chat_model" placeholder="例如 deepseek-chat / gpt-* / claude-*" />
            </el-form-item>
            <el-form-item label="Embedding Model">
              <el-input v-model="form.embedding_model" placeholder="OpenAI-compatible embedding 模型，可留空" />
            </el-form-item>
          </div>

          <div class="form-grid metric-columns">
            <el-form-item label="Temperature">
              <el-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" />
            </el-form-item>
            <el-form-item label="Max Tokens">
              <el-input-number v-model="form.max_tokens" :min="0" :step="256" />
            </el-form-item>
            <el-form-item label="Timeout">
              <el-input-number v-model="form.timeout_seconds" :min="5" :max="300" />
            </el-form-item>
          </div>

          <el-form-item label="扩展 JSON">
            <el-input
              v-model="form.extra_json"
              type="textarea"
              :rows="4"
              placeholder='例如 {"organization":"xxx"}，暂作为预留配置保存'
            />
          </el-form-item>

          <div class="switch-row">
            <el-checkbox v-model="form.enabled">启用配置</el-checkbox>
            <el-checkbox v-model="form.is_default">保存后设为默认</el-checkbox>
          </div>

          <div v-if="current?.last_test_status" class="test-result" :class="current.last_test_status">
            <strong>{{ current.last_test_status === "success" ? "最近测试通过" : "最近测试失败" }}</strong>
            <span>{{ current.last_test_message || "-" }}</span>
            <em>{{ timeText(current.last_test_at) }}</em>
          </div>

          <div class="submit-row">
            <el-button type="primary" :loading="saving" @click="saveConfig">
              <IconifyIconOnline icon="ri:save-3-line" />
              保存配置
            </el-button>
            <el-button @click="resetForm">
              <IconifyIconOnline icon="ri:restart-line" />
              清空
            </el-button>
          </div>
        </el-form>
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.ai-models-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.page-head,
.config-list,
.config-editor {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
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

.page-head h2 {
  margin: 0;
  color: var(--app-text);
  font-size: 22px;
  font-weight: 760;
}

.page-head p {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.head-stats {
  display: flex;
  align-items: center;
  gap: 12px;
}

.head-stats > div {
  min-width: 110px;
  padding: 10px 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.head-stats span,
.head-stats strong {
  display: block;
}

.head-stats span {
  color: var(--app-text-secondary);
  font-size: 12px;
}

.head-stats strong {
  margin-top: 4px;
  color: var(--app-text);
}

.model-layout {
  display: grid;
  grid-template-columns: minmax(260px, 360px) minmax(0, 1fr);
  gap: 16px;
}

.config-list {
  min-height: 520px;
  padding: 12px;
}

.config-item {
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid transparent;
  border-radius: 8px;
  cursor: pointer;
}

.config-item + .config-item {
  margin-top: 8px;
}

.config-item:hover,
.config-item.active {
  background: var(--app-surface-soft);
  border-color: var(--app-border);
}

.config-main strong,
.config-main span {
  display: block;
}

.config-main strong {
  color: var(--app-text);
}

.config-main span {
  margin-top: 4px;
  color: var(--app-text-secondary);
  font-size: 12px;
}

.config-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.config-editor {
  padding: 20px;
}

.editor-title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.editor-title h3 {
  margin: 0;
  color: var(--app-text);
  font-size: 18px;
}

.editor-title p {
  margin: 6px 0 0;
  color: var(--app-text-secondary);
}

.editor-actions,
.submit-row,
.switch-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

.model-form {
  max-width: 980px;
}

.form-grid {
  display: grid;
  gap: 12px;
}

.form-grid.two-columns {
  grid-template-columns: repeat(2, minmax(260px, 1fr));
}

.form-grid.metric-columns {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-grid :deep(.el-form-item) {
  min-width: 0;
}

.form-grid :deep(.el-form-item__label) {
  min-width: 118px;
  line-height: 1.25;
  white-space: normal;
}

.form-grid :deep(.el-input-number) {
  width: 100%;
}

.switch-row {
  margin: 4px 0 18px 118px;
}

.test-result {
  display: grid;
  gap: 4px;
  margin: 0 0 18px 118px;
  padding: 12px;
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface-soft);
}

.test-result.success {
  border-color: rgb(22 163 74 / 36%);
}

.test-result.failed {
  border-color: rgb(220 38 38 / 36%);
}

.test-result strong {
  color: var(--app-text);
}

.test-result span,
.test-result em {
  color: var(--app-text-secondary);
  font-style: normal;
}

.submit-row {
  margin-left: 118px;
}

@media (max-width: 1100px) {
  .page-head,
  .head-stats,
  .editor-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .model-layout,
  .form-grid {
    grid-template-columns: 1fr;
  }

  .switch-row,
  .test-result,
  .submit-row {
    margin-left: 0;
  }
}
</style>
