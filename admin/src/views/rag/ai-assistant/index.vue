<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import {
  askAIAssistant,
  getRAGIndexStats,
  rebuildRAGIndex,
  type AIAssistantResult,
  type RAGIndexStats
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGAIAssistant" });

const question = ref("帮我总结最近系统日志");
const loading = ref(false);
const rebuilding = ref(false);
const result = ref<AIAssistantResult>();
const ragStats = ref<RAGIndexStats>();

const examples = [
  "帮我总结最近系统日志",
  "分析最近异常",
  "生成用户操作报告",
  "今天谁登录失败最多？"
];

const rows = computed(() => result.value?.rows ?? []);
const sources = computed(() => result.value?.sources ?? []);
const sourceTypeLabels: Record<string, string> = {
  site_resource: "资源",
  site_project: "项目",
  site_tech_stack: "技术栈",
  site_timeline: "时间线",
  uploaded_document: "上传文档"
};
const columns = computed(() => {
  const first = rows.value[0];
  return first ? Object.keys(first) : [];
});

type AnswerBlock =
  | { type: "heading"; text: string }
  | { type: "paragraph"; html: string }
  | { type: "list"; items: string[] }
  | { type: "code"; language: string; code: string };

const answerTitle = computed(() => {
  const answer = result.value?.answer ?? "";
  const firstHeading = answer.split(/\r?\n/).find(line => /^#{1,4}\s+/.test(line.trim()));
  if (firstHeading) return firstHeading.replace(/^#{1,4}\s+/, "").trim();
  return question.value.trim() ? `\u56de\u7b54\uff1a${question.value.trim()}` : "\u5206\u6790\u7ed3\u8bba";
});

const answerBlocks = computed<AnswerBlock[]>(() => parseAnswerBlocks(result.value?.answer ?? ""));

const escapeHTML = (value: string) =>
  value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");

const inlineMarkdown = (value: string) =>
  escapeHTML(value)
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/`([^`]+)`/g, "<code>$1</code>");

function parseAnswerBlocks(markdown: string): AnswerBlock[] {
  const text = markdown.trim();
  if (!text) return [];
  const blocks: AnswerBlock[] = [];
  const fencePattern = /```([a-zA-Z0-9_-]*)\s*([\s\S]*?)```/g;
  let cursor = 0;
  let match: RegExpExecArray | null;
  while ((match = fencePattern.exec(text))) {
    pushAnswerTextBlocks(text.slice(cursor, match.index), blocks);
    blocks.push({ type: "code", language: match[1] || "text", code: match[2].trim() });
    cursor = match.index + match[0].length;
  }
  pushAnswerTextBlocks(text.slice(cursor), blocks);
  return blocks;
}

function pushAnswerTextBlocks(text: string, blocks: AnswerBlock[]) {
  const lines = text.split(/\r?\n/).map(line => line.trim()).filter(Boolean);
  let listItems: string[] = [];
  const flushList = () => {
    if (listItems.length) {
      blocks.push({ type: "list", items: listItems });
      listItems = [];
    }
  };
  lines.forEach(line => {
    const heading = line.match(/^#{1,4}\s+(.+)$/);
    if (heading) {
      flushList();
      blocks.push({ type: "heading", text: heading[1].trim() });
      return;
    }
    const list = line.match(/^[-*]\s+(.+)$/);
    if (list) {
      listItems.push(list[1].trim());
      return;
    }
    flushList();
    blocks.push({ type: "paragraph", html: inlineMarkdown(line) });
  });
  flushList();
}

const ask = async (text = question.value) => {
  question.value = text;
  loading.value = true;
  try {
    const res = await askAIAssistant(question.value);
    result.value = res.result;
  } catch {
    message("AI 助手分析失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const loadRAGStats = async () => {
  try {
    const res = await getRAGIndexStats();
    ragStats.value = res.stats;
  } catch {
    ragStats.value = undefined;
  }
};

const sourceTypeLabel = (type: string) => sourceTypeLabels[type] ?? type;

const rebuildIndex = async () => {
  rebuilding.value = true;
  try {
    await rebuildRAGIndex();
    await loadRAGStats();
    message("RAG 索引重建任务已提交", { type: "success" });
  } catch {
    message("RAG 索引重建任务提交失败", { type: "error" });
  } finally {
    rebuilding.value = false;
  }
};

onMounted(loadRAGStats);
</script>

<template>
  <div class="ai-page">
    <section class="ai-hero">
      <div>
        <p class="eyebrow">Admin copilot</p>
        <h2>AI 助手</h2>
        <p>用自然语言总结日志、分析异常、生成用户操作报告。</p>
      </div>
    </section>

    <section class="ask-panel">
      <div class="rag-status">
        <div>
          <strong>{{ ragStats?.total_chunks ?? 0 }}</strong>
          <span>chunks</span>
        </div>
        <div>
          <strong>{{ ragStats?.chat_enabled ? "on" : "off" }}</strong>
          <span>chat model</span>
        </div>
        <div>
          <strong>{{ ragStats?.top_k ?? "-" }}</strong>
          <span>top k</span>
        </div>
        <el-button :disabled="rebuilding" :loading="rebuilding" @click="rebuildIndex">重建索引</el-button>
      </div>
      <el-input
        v-model="question"
        type="textarea"
        :rows="3"
        resize="none"
        placeholder="例如：今天谁登录失败最多？"
      />
      <div class="ask-actions">
        <div class="example-list">
          <button v-for="item in examples" :key="item" type="button" @click="ask(item)">
            {{ item }}
          </button>
        </div>
        <el-button type="primary" :loading="loading" @click="ask()">开始分析</el-button>
      </div>
    </section>

    <section v-if="result" class="result-grid">
      <div class="result-card answer-card">
        <div class="answer-title">{{ answerTitle }}</div>
        <div class="assistant-answer">
          <template v-for="(block, index) in answerBlocks" :key="index">
            <h4 v-if="block.type === 'heading'">{{ block.text }}</h4>
            <ul v-else-if="block.type === 'list'">
              <li v-for="(item, itemIndex) in block.items" :key="itemIndex">{{ item }}</li>
            </ul>
            <figure v-else-if="block.type === 'code'" class="answer-code">
              <figcaption>{{ block.language }}</figcaption>
              <pre><code>{{ block.code }}</code></pre>
            </figure>
            <p v-else v-html="block.html" />
          </template>
        </div>
        <div class="card-title">分析结论</div>
        <p>{{ result.answer }}</p>
      </div>
      <div class="result-card">
        <div class="card-title">关键洞察</div>
        <ul class="insight-list">
          <li v-for="item in result.insights" :key="item">{{ item }}</li>
          <li v-if="!result.insights?.length">暂无洞察</li>
        </ul>
      </div>
      <div v-if="sources.length" class="result-card full">
        <div class="card-title">引用来源</div>
        <div class="source-list">
          <div v-for="item in sources" :key="`${item.source_type}-${item.source_id}-${item.title}`" class="source-item">
            <div>
              <strong>{{ item.title }}</strong>
              <span>{{ sourceTypeLabel(item.source_type) }} #{{ item.source_id }}</span>
              <p v-if="item.highlighted_text" v-html="item.highlighted_text" />
            </div>
            <em>{{ Math.round(item.score * 100) }}%</em>
          </div>
        </div>
      </div>
      <div class="result-card full">
        <div class="card-title">数据明细</div>
        <el-table v-if="rows.length" :data="rows" stripe>
          <el-table-column
            v-for="column in columns"
            :key="column"
            :prop="column"
            :label="column"
            min-width="150"
            show-overflow-tooltip
          />
        </el-table>
        <el-empty v-else description="暂无明细数据" />
      </div>
    </section>
  </div>
</template>

<style scoped lang="scss">
.ai-page {
  padding: 24px;
  display: grid;
  gap: 16px;
}

.ai-hero,
.ask-panel,
.result-card {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.ai-hero {
  padding: 22px 24px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-violet);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.ai-hero h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 760;
}

.ai-hero p:last-child {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.ask-panel {
  padding: 18px;
}

.rag-status {
  display: grid;
  grid-template-columns: repeat(3, minmax(90px, 1fr)) auto;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
}

.rag-status > div {
  padding: 10px 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.rag-status strong,
.rag-status span {
  display: block;
}

.rag-status strong {
  color: var(--app-text);
  font-size: 18px;
}

.rag-status span {
  margin-top: 2px;
  color: var(--app-text-secondary);
  font-size: 12px;
}

.ask-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 12px;
}

.example-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.example-list button {
  height: 30px;
  padding: 0 10px;
  color: var(--app-primary-strong);
  cursor: pointer;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 999px;
}

.result-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  align-items: start;
}

.result-card {
  padding: 18px;
}

.result-card.full {
  grid-column: 1 / -1;
}

.card-title {
  margin-bottom: 10px;
  color: var(--app-text);
  font-weight: 760;
}

.answer-card p {
  margin: 0;
  color: var(--app-text-secondary);
  line-height: 1.7;
}

.answer-card > .card-title,
.answer-card > p {
  display: none;
}

.answer-title {
  padding-bottom: 12px;
  margin-bottom: 14px;
  color: var(--app-text);
  font-size: 20px;
  font-weight: 760;
  line-height: 1.35;
  border-bottom: 1px solid var(--app-border);
  overflow-wrap: anywhere;
}

.assistant-answer {
  display: grid;
  gap: 12px;
}

.assistant-answer h4,
.assistant-answer p,
.assistant-answer ul,
.answer-code {
  margin: 0;
}

.assistant-answer h4 {
  color: var(--app-text);
  font-size: 16px;
  line-height: 1.4;
}

.assistant-answer p,
.assistant-answer li {
  color: var(--app-text-secondary);
  line-height: 1.75;
}

.assistant-answer ul {
  padding-left: 20px;
}

.assistant-answer :deep(code) {
  padding: 2px 6px;
  color: var(--app-primary-strong);
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  font-family: "Menlo", "Consolas", monospace;
  font-size: 0.92em;
}

.answer-code {
  overflow: hidden;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.answer-code figcaption {
  padding: 8px 12px;
  color: var(--app-text-secondary);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  border-bottom: 1px solid var(--app-border);
}

.answer-code pre {
  margin: 0;
  padding: 14px;
  overflow: auto;
}

.answer-code code {
  display: block;
  min-width: max-content;
  color: var(--app-text);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 13px;
  line-height: 1.7;
}

.insight-list {
  margin: 0;
  padding-left: 18px;
  color: var(--app-text-secondary);
  line-height: 1.8;
}

.source-list {
  display: grid;
  gap: 10px;
}

.source-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.source-item strong,
.source-item span {
  display: block;
}

.source-item strong {
  color: var(--app-text);
}

.source-item span,
.source-item em {
  margin-top: 4px;
  color: var(--app-text-secondary);
  font-style: normal;
  font-size: 12px;
}

.source-item p {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.source-item :deep(mark) {
  padding: 0 3px;
  color: var(--app-text);
  background: rgb(245 158 11 / 22%);
  border-radius: 4px;
}

@media (max-width: 900px) {
  .rag-status,
  .ask-actions,
  .result-grid {
    align-items: flex-start;
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}
</style>
