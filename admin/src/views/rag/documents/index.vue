<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import dayjs from "dayjs";
import { ElMessageBox, type UploadRequestOptions } from "element-plus";
import {
  deleteDocument,
  getUploadedDocuments,
  previewDocument,
  rebuildDocument,
  uploadDocument,
  type UploadedDocument
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "RAGDocuments" });

const loading = ref(false);
const uploading = ref(false);
const rebuildingId = ref<number>();
const previewVisible = ref(false);
const previewLoading = ref(false);
const previewItem = ref<UploadedDocument>();
const documents = ref<UploadedDocument[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(10);

const activeCount = computed(
  () => documents.value.filter(item => item.status === "active").length
);
const failedCount = computed(
  () => documents.value.filter(item => item.status === "failed").length
);
const totalChunks = computed(() =>
  documents.value.reduce((sum, item) => sum + (item.chunk_count || 0), 0)
);

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";

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
    const res = await uploadDocument(file);
    options.onSuccess?.(res);
    message("文档已上传并写入 RAG 索引", { type: "success" });
    page.value = 1;
    await loadDocuments();
  } catch (error: any) {
    const errorMessage = error?.response?.data?.error || error?.message || "上传失败";
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
        <h2>文档管理</h2>
        <p>上传 PDF、Markdown、TXT 文档，解析文本后写入 RAG 知识索引。</p>
      </div>
      <el-button @click="loadDocuments">
        <IconifyIconOnline icon="ri:refresh-line" />
        刷新
      </el-button>
    </section>

    <section class="document-layout">
      <aside class="side-panel">
        <div class="upload-box">
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
            <span>支持 PDF / MD / TXT，单个文件不超过 20MB</span>
          </el-upload>
        </div>

        <div class="summary-grid">
          <div>
            <span>当前页已入库</span>
            <strong>{{ activeCount }}</strong>
          </div>
          <div>
            <span>当前页失败</span>
            <strong>{{ failedCount }}</strong>
          </div>
          <div>
            <span>当前页片段</span>
            <strong>{{ totalChunks }}</strong>
          </div>
        </div>

        <div class="note-panel">
          <h3>解析说明</h3>
          <p>Markdown 和 TXT 会直接提取全文。PDF 当前采用轻量文本提取，扫描件或复杂排版 PDF 可能需要转成文本版后重新上传。</p>
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
          <el-table-column label="文档" min-width="260" show-overflow-tooltip>
            <template #default="{ row }">
              <div class="doc-name">
                <IconifyIconOnline icon="ri:file-text-line" />
                <div>
                  <strong>{{ row.original_name }}</strong>
                  <span>{{ row.mime_type || "unknown" }}</span>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="大小" width="110">
            <template #default="{ row }">{{ fileSizeText(row.file_size) }}</template>
          </el-table-column>
          <el-table-column label="片段" width="90" prop="chunk_count" />
          <el-table-column label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)" effect="light">
                {{ statusText(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="180">
            <template #default="{ row }">{{ timeText(row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="失败原因" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.error_message || "-" }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button size="small" @click="showPreview(row)">预览</el-button>
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

    <el-drawer v-model="previewVisible" size="52%" title="文档预览">
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
  grid-template-columns: minmax(280px, 360px) minmax(0, 1fr);
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

.upload-box :deep(.el-upload),
.upload-box :deep(.el-upload-dragger) {
  width: 100%;
}

.upload-box :deep(.el-upload-dragger) {
  display: grid;
  place-items: center;
  gap: 8px;
  min-height: 190px;
  border-radius: 8px;
}

.upload-icon {
  color: var(--app-primary);
  font-size: 34px;
}

.upload-box strong {
  color: var(--app-text);
  font-size: 16px;
}

.upload-box span,
.summary-grid span,
.panel-title span,
.doc-name span,
.note-panel p {
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
  min-height: 84px;
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

.panel-title {
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

.preview-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.preview-meta h3 {
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

.preview-panel pre {
  min-height: 360px;
  max-height: calc(100vh - 220px);
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

@media (max-width: 1100px) {
  .document-head,
  .panel-title {
    align-items: flex-start;
    flex-direction: column;
  }

  .document-layout {
    grid-template-columns: 1fr;
  }
}
</style>
