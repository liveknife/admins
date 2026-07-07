<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import { ElMessageBox } from "element-plus";
import {
  deleteRAGEvalCase,
  getRAGEvalCases,
  getRAGEvalRuns,
  runRAGEval,
  saveRAGEvalCase,
  type RAGEvalCase,
  type RAGEvalRun,
  type RAGEvalRunSummary
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGEvals" });

const loading = ref(false);
const running = ref(false);
const dialogVisible = ref(false);
const editingId = ref("");
const includeInternal = ref(true);
const cases = ref<RAGEvalCase[]>([]);
const runs = ref<RAGEvalRunSummary[]>([]);
const currentRun = ref<RAGEvalRun>();

const form = reactive({
  id: "",
  question: "",
  expected_sources: "",
  expected_terms: "",
  enabled: true
});

const enabledCount = computed(() => cases.value.filter(item => item.enabled).length);
const recallRate = computed(() => {
  if (!currentRun.value?.total) return 0;
  return Math.round((currentRun.value.recall_hits / currentRun.value.total) * 100);
});

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";
const percent = (value = 0) => `${Math.round(value * 100)}%`;
const splitList = (value: string) =>
  value
    .split(/[,\n]/)
    .map(item => item.trim())
    .filter(Boolean);

const loadData = async () => {
  loading.value = true;
  try {
    const [caseRes, runRes] = await Promise.all([
      getRAGEvalCases(),
      getRAGEvalRuns(20)
    ]);
    cases.value = caseRes.cases ?? [];
    runs.value = runRes.runs ?? [];
  } catch {
    message("RAG 评测数据加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const openCreate = () => {
  editingId.value = "";
  Object.assign(form, {
    id: "",
    question: "",
    expected_sources: "",
    expected_terms: "",
    enabled: true
  });
  dialogVisible.value = true;
};

const openEdit = (row: RAGEvalCase) => {
  editingId.value = row.id;
  Object.assign(form, {
    id: row.id,
    question: row.question,
    expected_sources: (row.expected_sources ?? []).join(", "),
    expected_terms: (row.expected_terms ?? []).join(", "),
    enabled: row.enabled
  });
  dialogVisible.value = true;
};

const submitCase = async () => {
  const payload: RAGEvalCase = {
    id: form.id.trim(),
    question: form.question.trim(),
    expected_sources: splitList(form.expected_sources),
    expected_terms: splitList(form.expected_terms),
    enabled: form.enabled
  };
  if (!payload.question) {
    message("请输入评测问题", { type: "warning" });
    return;
  }
  try {
    await saveRAGEvalCase(payload, editingId.value || undefined);
    message("评测用例已保存", { type: "success" });
    dialogVisible.value = false;
    await loadData();
  } catch {
    message("评测用例保存失败", { type: "error" });
  }
};

const removeCase = async (row: RAGEvalCase) => {
  try {
    await ElMessageBox.confirm(`删除评测用例：${row.id}`, "删除评测用例", {
      type: "warning",
      confirmButtonText: "删除",
      cancelButtonText: "取消"
    });
    await deleteRAGEvalCase(row.id);
    message("评测用例已删除", { type: "success" });
    await loadData();
  } catch (error) {
    if (error !== "cancel") message("评测用例删除失败", { type: "error" });
  }
};

const executeEval = async () => {
  running.value = true;
  try {
    const res = await runRAGEval(includeInternal.value);
    currentRun.value = res.run;
    message("RAG 评测已完成", { type: "success" });
    await loadData();
  } catch {
    message("RAG 评测运行失败", { type: "error" });
  } finally {
    running.value = false;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="eval-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Evaluation set</p>
        <h2>RAG 评测中心</h2>
        <p>维护固定问题集，跑召回率、命中来源和回答质量，避免后续调参把效果改差。</p>
      </div>
      <div class="head-actions">
        <el-switch
          v-model="includeInternal"
          active-text="含 Internal"
          inactive-text="仅 Public"
          inline-prompt
        />
        <el-button @click="loadData">刷新</el-button>
        <el-button type="primary" :loading="running" @click="executeEval">运行评测</el-button>
      </div>
    </section>

    <section class="metric-grid">
      <div class="metric-card">
        <span>评测用例</span>
        <strong>{{ cases.length }}</strong>
        <small>{{ enabledCount }} 个启用</small>
      </div>
      <div class="metric-card">
        <span>本次召回率</span>
        <strong>{{ recallRate }}%</strong>
        <small>{{ currentRun?.recall_hits ?? 0 }} / {{ currentRun?.total ?? 0 }}</small>
      </div>
      <div class="metric-card">
        <span>平均质量</span>
        <strong>{{ percent(currentRun?.average_quality ?? 0) }}</strong>
        <small>命中词覆盖率</small>
      </div>
      <div class="metric-card">
        <span>平均耗时</span>
        <strong>{{ currentRun?.average_latency_ms ?? 0 }}ms</strong>
        <small>{{ timeText(currentRun?.created_at) }}</small>
      </div>
    </section>

    <section class="layout">
      <div class="panel">
        <div class="panel-title">
          <h3>评测用例</h3>
          <el-button type="primary" plain @click="openCreate">新增用例</el-button>
        </div>
        <el-table :data="cases" stripe>
          <el-table-column label="问题" min-width="260">
            <template #default="{ row }">
              <strong>{{ row.id }}</strong>
              <p>{{ row.question }}</p>
            </template>
          </el-table-column>
          <el-table-column label="期望来源" min-width="180">
            <template #default="{ row }">
              <el-tag
                v-for="item in row.expected_sources"
                :key="item"
                size="small"
                effect="light"
              >
                {{ item }}
              </el-tag>
              <span v-if="!row.expected_sources?.length">任意来源</span>
            </template>
          </el-table-column>
          <el-table-column label="期望词" min-width="180">
            <template #default="{ row }">
              <span>{{ row.expected_terms?.join(" / ") || "-" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'info'" effect="light">
                {{ row.enabled ? "启用" : "停用" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="openEdit(row)">编辑</el-button>
              <el-button size="small" type="danger" plain @click="removeCase(row)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel">
        <div class="panel-title">
          <h3>运行历史</h3>
          <span>最近 20 次</span>
        </div>
        <el-table :data="runs" stripe height="320">
          <el-table-column label="ID" width="80">
            <template #default="{ row }">#{{ row.id }}</template>
          </el-table-column>
          <el-table-column label="召回" width="110">
            <template #default="{ row }">{{ row.recall_hits }} / {{ row.total }}</template>
          </el-table-column>
          <el-table-column label="质量" width="90">
            <template #default="{ row }">{{ percent(row.average_quality) }}</template>
          </el-table-column>
          <el-table-column label="耗时" width="110">
            <template #default="{ row }">{{ row.average_latency_ms }}ms</template>
          </el-table-column>
          <el-table-column label="时间" min-width="150">
            <template #default="{ row }">{{ timeText(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </div>

      <div class="panel full">
        <div class="panel-title">
          <h3>本次结果</h3>
          <span>来源命中和答案质量</span>
        </div>
        <el-table :data="currentRun?.results ?? []" stripe>
          <el-table-column label="用例" min-width="260">
            <template #default="{ row }">
              <strong>{{ row.case.id }}</strong>
              <p>{{ row.case.question }}</p>
            </template>
          </el-table-column>
          <el-table-column label="召回" width="90">
            <template #default="{ row }">
              <el-tag :type="row.recall_hit ? 'success' : 'danger'" effect="light">
                {{ row.recall_hit ? "命中" : "偏离" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="质量" width="100">
            <template #default="{ row }">{{ percent(row.answer_quality) }}</template>
          </el-table-column>
          <el-table-column label="Top score" width="110">
            <template #default="{ row }">{{ percent(row.top_score) }}</template>
          </el-table-column>
          <el-table-column label="答案" min-width="320" show-overflow-tooltip>
            <template #default="{ row }">{{ row.answer }}</template>
          </el-table-column>
        </el-table>
      </div>
    </section>

    <el-dialog v-model="dialogVisible" title="评测用例" width="640px">
      <el-form label-position="top">
        <el-form-item label="ID">
          <el-input v-model="form.id" :disabled="Boolean(editingId)" placeholder="不填自动生成" />
        </el-form-item>
        <el-form-item label="问题">
          <el-input v-model="form.question" type="textarea" :rows="3" resize="none" />
        </el-form-item>
        <el-form-item label="期望来源">
          <el-input v-model="form.expected_sources" placeholder="site_project, uploaded_document" />
        </el-form-item>
        <el-form-item label="期望词">
          <el-input v-model="form.expected_terms" placeholder="Go, Vue, 项目" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitCase">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.eval-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.page-head,
.metric-card,
.panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-primary);
  font-family: Menlo, Consolas, monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.page-head h2,
.panel-title h3 {
  margin: 0;
  color: var(--app-text);
}

.page-head p,
.panel-title span,
.metric-card span,
.metric-card small,
.el-table p {
  color: var(--app-text-secondary);
}

.page-head p,
.el-table p {
  margin: 8px 0 0;
}

.head-actions,
.panel-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card {
  display: grid;
  gap: 6px;
  padding: 18px;
}

.metric-card strong {
  color: var(--app-text);
  font-size: 26px;
}

.layout {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(320px, 0.8fr);
  gap: 16px;
}

.panel {
  min-width: 0;
  padding: 18px;
}

.panel.full {
  grid-column: 1 / -1;
}

.panel-title {
  margin-bottom: 14px;
}

.el-tag + .el-tag {
  margin-left: 6px;
}

.el-table strong {
  color: var(--app-text);
}

@media (max-width: 1100px) {
  .page-head,
  .head-actions,
  .panel-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .metric-grid,
  .layout {
    grid-template-columns: 1fr;
  }
}
</style>
