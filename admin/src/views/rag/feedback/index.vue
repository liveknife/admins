<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import {
  convertRAGFeedbackToEvalCase,
  getRAGFeedback,
  updateRAGFeedbackStatus,
  type RAGFeedback
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGFeedback" });

const loading = ref(false);
const handlingId = ref<number>();
const rating = ref("");
const status = ref("");
const feedback = ref<RAGFeedback[]>([]);

const openCount = computed(
  () => feedback.value.filter(item => item.status === "open").length
);
const negativeCount = computed(
  () => feedback.value.filter(item => item.rating === "down").length
);

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";

const ratingType = (value: string) => {
  if (value === "up") return "success";
  if (value === "down") return "danger";
  return "info";
};

const statusType = (value: string) => {
  if (value === "resolved") return "success";
  if (value === "ignored") return "info";
  if (value === "triaged") return "warning";
  return "danger";
};

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getRAGFeedback({
      limit: 100,
      rating: rating.value || undefined,
      status: status.value || undefined
    });
    feedback.value = res.feedback ?? [];
  } catch {
    message("RAG 反馈加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const setStatus = async (row: RAGFeedback, nextStatus: string) => {
  handlingId.value = row.id;
  try {
    const res = await updateRAGFeedbackStatus(row.id, {
      status: nextStatus,
      admin_note: row.admin_note
    });
    Object.assign(row, res.feedback);
    message("反馈状态已更新", { type: "success" });
  } catch {
    message("反馈状态更新失败", { type: "error" });
  } finally {
    handlingId.value = undefined;
  }
};

const convertToEval = async (row: RAGFeedback) => {
  handlingId.value = row.id;
  try {
    const res = await convertRAGFeedbackToEvalCase(row.id);
    row.status = "resolved";
    row.converted_eval_case_id = res.case.id;
    message(`已转为评测用例：${res.case.id}`, { type: "success" });
  } catch {
    message("反馈转评测用例失败", { type: "error" });
  } finally {
    handlingId.value = undefined;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="feedback-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Feedback loop</p>
        <h2>反馈处理中心</h2>
        <p>处理官网知识库问答反馈，把低质量问题沉淀为评测用例，让后续优化有回归基准。</p>
      </div>
      <div class="head-actions">
        <el-select v-model="rating" clearable placeholder="评分" style="width: 120px" @change="loadData">
          <el-option label="赞" value="up" />
          <el-option label="踩" value="down" />
          <el-option label="中立" value="neutral" />
        </el-select>
        <el-select v-model="status" clearable placeholder="状态" style="width: 130px" @change="loadData">
          <el-option label="Open" value="open" />
          <el-option label="Triaged" value="triaged" />
          <el-option label="Resolved" value="resolved" />
          <el-option label="Ignored" value="ignored" />
        </el-select>
        <el-button @click="loadData">刷新</el-button>
      </div>
    </section>

    <section class="metric-grid">
      <div class="metric-card">
        <span>反馈总数</span>
        <strong>{{ feedback.length }}</strong>
        <small>当前筛选范围</small>
      </div>
      <div class="metric-card">
        <span>待处理</span>
        <strong>{{ openCount }}</strong>
        <small>建议优先查看差评</small>
      </div>
      <div class="metric-card">
        <span>差评</span>
        <strong>{{ negativeCount }}</strong>
        <small>适合转评测用例</small>
      </div>
    </section>

    <section class="panel">
      <el-table :data="feedback" stripe>
        <el-table-column label="反馈" min-width="320">
          <template #default="{ row }">
            <div class="question-cell">
              <strong>{{ row.question || "未记录问题" }}</strong>
              <span>{{ row.comment || "无补充说明" }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="评分" width="90">
          <template #default="{ row }">
            <el-tag :type="ratingType(row.rating)" effect="light">{{ row.rating }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" effect="light">{{ row.status || "open" }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="备注" min-width="220">
          <template #default="{ row }">
            <el-input
              v-model="row.admin_note"
              size="small"
              placeholder="处理备注"
              @change="setStatus(row, row.status || 'triaged')"
            />
          </template>
        </el-table-column>
        <el-table-column label="评测用例" width="150">
          <template #default="{ row }">
            <el-tag v-if="row.converted_eval_case_id" type="success" effect="light">
              {{ row.converted_eval_case_id }}
            </el-tag>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ timeText(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button
              size="small"
              :loading="handlingId === row.id"
              @click="setStatus(row, 'triaged')"
            >
              标记
            </el-button>
            <el-button
              size="small"
              type="success"
              plain
              :loading="handlingId === row.id"
              @click="setStatus(row, 'resolved')"
            >
              解决
            </el-button>
            <el-button
              size="small"
              type="warning"
              plain
              :disabled="Boolean(row.converted_eval_case_id)"
              :loading="handlingId === row.id"
              @click="convertToEval(row)"
            >
              转评测
            </el-button>
            <el-button
              size="small"
              type="info"
              plain
              :loading="handlingId === row.id"
              @click="setStatus(row, 'ignored')"
            >
              忽略
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>

<style scoped lang="scss">
.feedback-page {
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

.page-head h2 {
  margin: 0;
  color: var(--app-text);
}

.page-head p,
.metric-card span,
.metric-card small,
.question-cell span {
  color: var(--app-text-secondary);
}

.page-head p {
  margin: 8px 0 0;
}

.head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
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

.panel {
  min-width: 0;
  padding: 18px;
}

.question-cell {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.question-cell strong,
.question-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 1100px) {
  .page-head,
  .head-actions {
    align-items: flex-start;
    flex-direction: column;
  }

  .metric-grid {
    grid-template-columns: 1fr;
  }
}
</style>
