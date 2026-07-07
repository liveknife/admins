<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import { ElMessageBox, type UploadRequestOptions } from "element-plus";
import {
  deleteDocument,
  getDocumentChunks,
  getUploadedDocuments,
  previewDocument,
  rebuildDocument,
  updateDocumentVisibility,
  uploadDocument,
  type KnowledgeChunkPreview,
  type UploadedDocument
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGDocuments" });

const loading = ref(false);
const uploading = ref(false);
const rebuildingId = ref<number>();
const switchingId = ref<number>();
const previewVisible = ref(false);
const previewLoading = ref(false);
const previewItem = ref<UploadedDocument>();
const chunkVisible = ref(false);
const chunkLoading = ref(false);
const chunkDocument = ref<UploadedDocument>();
const chunks = ref<KnowledgeChunkPreview[]>([]);
const activeChunk = ref<KnowledgeChunkPreview>();
const uploadVisibility = ref("internal");
const documents = ref<UploadedDocument[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);

const activeCount = computed(
  () => documents.value.filter(item => item.status === "active").length
);
const publicCount = computed(
  () => documents.value.filter(item => item.visibility === "public").length
);
const totalChunks = computed(() =>
  documents.value.reduce((sum, item) => sum + (item.chunk_count || 0), 0)
);

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm") : "-";

const fileSizeText = (size = 0) => {
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`;
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${size} B`;
};

const statusType = (status: string) => {
  if (status === "active") return "success";
  if (status === "failed") return "danger";
  return "info";
};

const statusText = (status: string) => {
  if (status === "active") return "已入库";
  if (status === "failed") return "失败";
  return status || "未知";
};

const visibilityText = (value: string) =>
  value === "public" ? "Public" : "Internal";

const loadDocuments = async () => {
  loading.value = true;
  try {
    const res = await getUploadedDocuments({
      page: page.value,
      page_size: pageSize.value
    });
    documents.value = res.documents ?? [];
    total.value = res.total ?? 0;
  } catch {
    message("文档列表加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const handleUpload = async (options: UploadRequestOptions) => {
  const file = options.file as File;
  uploading.value = true;
  try {
    const res = await uploadDocument(file, uploadVisibility.value);
    options.onSuccess?.(res);
    message("文档已上传并写入 RAG 索引", { type: "success" });
    page.value = 1;
    await loadDocuments();
  } catch (error: any) {
    const errorMessage =
      error?.response?.data?.error || error?.message || "上传失败";
    options.onError?.(error);
    message(errorMessage, { type: "error" });
  } finally {
    uploading.value = false;
  }
};

const showPreview = async (row: UploadedDocument) => {
  previewVisible.value = true;
  previewLoading.value = true;
  previewItem.value = row;
  try {
    const res = await previewDocument(row.id);
    previewItem.value = res.document;
  } catch {
    message("文档预览加载失败", { type: "error" });
  } finally {
    previewLoading.value = false;
  }
};

const showChunks = async (row: UploadedDocument) => {
  chunkVisible.value = true;
  chunkLoading.value = true;
  chunkDocument.value = row;
  chunks.value = [];
  activeChunk.value = undefined;
  try {
    const res = await getDocumentChunks(row.id, 200);
    chunks.value = res.chunks ?? [];
    activeChunk.value = chunks.value[0];
  } catch {
    message("chunk 预览加载失败", { type: "error" });
  } finally {
    chunkLoading.value = false;
  }
};

const submitRebuild = async (row: UploadedDocument) => {
  rebuildingId.value = row.id;
  try {
    await rebuildDocument(row.id);
    message("文档索引已重建", { type: "success" });
    await loadDocuments();
  } catch {
    message("文档索引重建失败", { type: "error" });
  } finally {
    rebuildingId.value = undefined;
  }
};

const changeVisibility = async (row: UploadedDocument, value: string) => {
  const previous = row.visibility;
  switchingId.value = row.id;
  try {
    const res = await updateDocumentVisibility(row.id, value);
    Object.assign(row, res.document);
    message(`已切换为 ${visibilityText(value)}`, { type: "success" });
  } catch {
    row.visibility = previous;
    message("可见性切换失败", { type: "error" });
  } finally {
    switchingId.value = undefined;
  }
};

const removeDocument = async (row: UploadedDocument) => {
  try {
    await ElMessageBox.confirm(
      `删除后会同时移除该文档的 RAG 片段：${row.original_name}`,
      "删除文档",
      { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
    );
    await deleteDocument(row.id);
    message("文档已删除", { type: "success" });
    await loadDocuments();
  } catch (error) {
    if (error !== "cancel") message("文档删除失败", { type: "error" });
  }
};

const handlePageChange = (value: number) => {
  page.value = value;
  loadDocuments();
};

const handleSizeChange = (value: number) => {
  pageSize.value = value;
  page.value = 1;
  loadDocuments();
};

onMounted(loadDocuments);
</script>

<template>
  <div class="document-page">
    <section class="document-head">
      <div>
        <p class="eyebrow">Knowledge files</p>
        <h2>RAG 文档管理</h2>
        <p>管理上传文档、公开范围和索引片段，排查官网问答能否命中正确资料。</p>
      </div>
      <el-button @click="loadDocuments">
        <IconifyIconOnline icon="ri:refresh-line" />
        刷新
      </el-button>
    </section>

    <section class="document-layout">
      <aside class="side-panel">
        <div class="upload-card">
          <div class="upload-mode">
            <span>上传范围</span>
            <el-radio-group v-model="uploadVisibility" size="small">
              <el-radio-button label="internal">Internal</el-radio-button>
              <el-radio-button label="public">Public</el-radio-button>
            </el-radio-group>
          </div>
          <el-upload
            drag
            action="#"
            :show-file-list="false"
            :http-request="handleUpload"
            :disabled="uploading"
            accept=".pdf,.md,.markdown,.txt"
          >
            <IconifyIconOnline icon="ri:file-upload-line" class="upload-icon" />
            <strong>{{ uploading ? "正在上传..." : "拖拽文档到这里" }}</strong>
            <span>PDF / Markdown / TXT，单文件 20MB 以内</span>
          </el-upload>
        </div>

        <div class="summary-grid">
          <div>
            <span>已入库</span>
            <strong>{{ activeCount }}</strong>
          </div>
          <div>
            <span>Public</span>
            <strong>{{ publicCount }}</strong>
          </div>
          <div>
            <span>Chunks</span>
            <strong>{{ totalChunks }}</strong>
          </div>
        </div>

        <div class="note-panel">
          <h3>范围说明</h3>
          <p>Public 会进入官网公开问答；Internal 只允许后台 AI 助手检索。切换范围会同步重建该文档的 chunk。</p>
        </div>
      </aside>

      <main class="list-panel" v-loading="loading">
        <div class="panel-title">
          <div>
            <h3>文档列表</h3>
            <span>共 {{ total }} 个文档</span>
          </div>
        </div>

        <el-table :data="documents" stripe>
          <el-table-column label="文档" min-width="280" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="doc-name">
                <IconifyIconOnline icon="ri:file-text-line" />
                <div>
                  <strong>{{ row.original_name }}</strong>
                  <span>{{ row.mime_type || "unknown" }} · {{ fileSizeText(row.file_size) }}</span>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="范围" width="150">
            <template #default="{ row }">
              <el-switch
                v-model="row.visibility"
                :loading="switchingId === row.id"
                active-value="public"
                inactive-value="internal"
                active-text="Public"
                inactive-text="Internal"
                inline-prompt
                @change="value => changeVisibility(row, String(value))"
              />
            </template>
          </el-table-column>
          <el-table-column label="片段" width="90" prop="chunk_count" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)" effect="light">
                {{ statusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="160">
            <template #default="{ row }">{{ timeText(row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="错误" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.error_message || "-" }}</template>
          </el-table-column>
          <el-table-column label="操作" width="270" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="showPreview(row)">原文</el-button>
              <el-button size="small" @click="showChunks(row)">Chunks</el-button>
              <el-button
                size="small"
                :loading="rebuildingId === row.id"
                @click="submitRebuild(row)"
              >
                重建
              </el-button>
              <el-button size="small" type="danger" plain @click="removeDocument(row)">
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination-row">
          <el-pagination
            background
            layout="total, sizes, prev, pager, next"
            :total="total"
            :current-page="page"
            :page-size="pageSize"
            :page-sizes="[10, 20, 50, 100]"
            @current-change="handlePageChange"
            @size-change="handleSizeChange"
          />
        </div>
      </main>
    </section>

    <el-drawer v-model="previewVisible" size="52%" title="文档原文">
      <div v-loading="previewLoading" class="preview-panel">
        <template v-if="previewItem">
          <div class="preview-meta">
            <h3>{{ previewItem.original_name }}</h3>
            <el-tag :type="statusType(previewItem.status)" effect="light">
              {{ statusText(previewItem.status) }}
            </el-tag>
          </div>
          <div v-if="previewItem.error_message" class="error-box">
            {{ previewItem.error_message }}
          </div>
          <pre>{{ previewItem.text_content || "暂无可预览文本" }}</pre>
        </template>
      </div>
    </el-drawer>

    <el-drawer v-model="chunkVisible" size="64%" title="Chunk 预览">
      <div v-loading="chunkLoading" class="chunk-layout">
        <aside class="chunk-list">
          <div class="chunk-doc">
            <strong>{{ chunkDocument?.original_name }}</strong>
            <span>{{ chunks.length }} chunks</span>
          </div>
          <button
            v-for="item in chunks"
            :key="item.id"
            type="button"
            :class="{ active: activeChunk?.id === item.id }"
            @click="activeChunk = item"
          >
            <strong>#{{ item.id }} {{ item.title }}</strong>
            <span>{{ item.token_count }} tokens · {{ item.visibility }}</span>
          </button>
          <el-empty v-if="!chunks.length" description="暂无 chunk" />
        </aside>
        <main class="chunk-reader">
          <template v-if="activeChunk">
            <div class="chunk-reader-head">
              <div>
                <h3>{{ activeChunk.title }}</h3>
                <span>{{ activeChunk.source_type }} #{{ activeChunk.source_id }}</span>
              </div>
              <el-tag effect="light">{{ activeChunk.visibility }}</el-tag>
            </div>
            <pre>{{ activeChunk.content }}</pre>
          </template>
          <el-empty v-else description="选择左侧 chunk 查看内容" />
        </main>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped lang="scss">
.document-page {
  display: grid;
  gap: 16px;
  padding: 24px;
}

.document-head,
.side-panel,
.list-panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.document-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 22px 24px;
}

.document-head h2,
.panel-title h3,
.note-panel h3 {
  margin: 0;
  color: var(--app-text);
}

.document-head p {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-primary);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.document-layout {
  display: grid;
  align-items: start;
  grid-template-columns: minmax(300px, 380px) minmax(0, 1fr);
  gap: 16px;
}

.side-panel,
.list-panel {
  min-width: 0;
  padding: 18px;
}

.side-panel {
  display: grid;
  gap: 16px;
}

.upload-card {
  display: grid;
  gap: 12px;
}

.upload-mode {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--app-text-secondary);
}

.upload-card :deep(.el-upload),
.upload-card :deep(.el-upload-dragger) {
  width: 100%;
}

.upload-card :deep(.el-upload-dragger) {
  display: grid;
  place-items: center;
  gap: 8px;
  min-height: 178px;
  border-radius: 8px;
}

.upload-icon {
  color: var(--app-primary);
  font-size: 34px;
}

.upload-card strong {
  color: var(--app-text);
  font-size: 16px;
}

.upload-card span,
.summary-grid span,
.panel-title span,
.doc-name span,
.note-panel p,
.chunk-list span,
.chunk-reader-head span {
  color: var(--app-text-secondary);
}

.summary-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.summary-grid > div {
  display: grid;
  gap: 8px;
  min-height: 78px;
  padding: 12px;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.summary-grid strong {
  color: var(--app-text);
  font-size: 24px;
}

.note-panel {
  display: grid;
  gap: 8px;
  padding: 14px;
  background: rgb(14 165 233 / 8%);
  border: 1px solid rgb(14 165 233 / 22%);
  border-radius: 8px;
}

.note-panel p {
  margin: 0;
  line-height: 1.7;
}

.panel-title,
.preview-meta,
.chunk-reader-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.doc-name {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.doc-name > svg {
  flex: 0 0 auto;
  color: var(--app-primary);
  font-size: 20px;
}

.doc-name div {
  display: grid;
  min-width: 0;
}

.doc-name strong,
.doc-name span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pagination-row {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}

.preview-panel {
  min-height: 280px;
}

.preview-meta h3,
.chunk-reader-head h3 {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
}

.error-box {
  margin-bottom: 12px;
  padding: 12px;
  color: var(--el-color-danger);
  white-space: pre-wrap;
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: 8px;
}

.preview-panel pre,
.chunk-reader pre {
  padding: 16px;
  margin: 0;
  overflow: auto;
  color: var(--app-text);
  line-height: 1.7;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.preview-panel pre {
  min-height: 360px;
  max-height: calc(100vh - 220px);
}

.chunk-layout {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 16px;
  min-height: calc(100vh - 120px);
}

.chunk-list {
  display: grid;
  align-content: start;
  gap: 10px;
  min-width: 0;
}

.chunk-doc,
.chunk-list button {
  display: grid;
  gap: 4px;
  padding: 12px;
  text-align: left;
  background: var(--app-surface-soft);
  border: 1px solid var(--app-border);
  border-radius: 8px;
}

.chunk-list button {
  cursor: pointer;
}

.chunk-list button.active {
  border-color: var(--app-primary);
  box-shadow: inset 3px 0 0 var(--app-primary);
}

.chunk-list strong {
  color: var(--app-text);
  overflow-wrap: anywhere;
}

.chunk-reader {
  min-width: 0;
}

.chunk-reader pre {
  max-height: calc(100vh - 210px);
}

@media (max-width: 1100px) {
  .document-head,
  .panel-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .document-layout,
  .chunk-layout {
    grid-template-columns: 1fr;
  }
}
</style>
