<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  getDatabaseCatalog,
  getDatabaseColumns,
  getDatabaseTables,
  type DatabaseCatalog,
  type DatabaseColumn,
  type DatabaseTable
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "SystemToolsDatabase" });

const loading = ref(false);
const dialogLoading = ref(false);
const dialogVisible = ref(false);
const catalog = ref<DatabaseCatalog>();
const tables = ref<DatabaseTable[]>([]);
const columns = ref<DatabaseColumn[]>([]);
const currentTable = ref<DatabaseTable>();

const query = reactive({
  database: "",
  table: "",
  engine: "",
  comment: ""
});

const databaseOptions = computed(() => catalog.value?.databases ?? []);
const engineOptions = computed(() => catalog.value?.engines ?? []);
const tableTotal = computed(() => tables.value.length);
const totalRows = computed(() =>
  tables.value.reduce((sum, item) => sum + Number(item.rows || 0), 0)
);

const loadCatalog = async () => {
  const res = await getDatabaseCatalog();
  catalog.value = res.catalog;
  query.database = res.catalog.current_database;
};

const loadTables = async () => {
  loading.value = true;
  try {
    if (!catalog.value) await loadCatalog();
    const res = await getDatabaseTables({
      database: query.database,
      table: query.table || undefined,
      engine: query.engine || undefined,
      comment: query.comment || undefined
    });
    tables.value = res.tables ?? [];
  } catch {
    message("数据库表加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const resetQuery = () => {
  query.database = catalog.value?.current_database ?? "";
  query.table = "";
  query.engine = "";
  query.comment = "";
  loadTables();
};

const openStructure = async (row: DatabaseTable) => {
  currentTable.value = row;
  dialogVisible.value = true;
  dialogLoading.value = true;
  try {
    const res = await getDatabaseColumns(row.name, query.database);
    columns.value = res.columns ?? [];
  } catch {
    message("表结构加载失败", { type: "error" });
    columns.value = [];
  } finally {
    dialogLoading.value = false;
  }
};

const createdAtText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD HH:mm:ss") : "-";

onMounted(loadTables);
</script>

<template>
  <div class="database-page">
    <section class="database-head">
      <div>
        <p class="eyebrow">Schema console</p>
        <h2>数据库表结构</h2>
        <p>查看当前连接数据库的表清单、数据行数、索引大小和字段结构。</p>
      </div>
      <div class="head-metrics">
        <div>
          <span>当前数据库</span>
          <strong>{{ query.database || "-" }}</strong>
        </div>
        <div>
          <span>表数量</span>
          <strong>{{ tableTotal }}</strong>
        </div>
        <div>
          <span>估算行数</span>
          <strong>{{ totalRows }}</strong>
        </div>
      </div>
    </section>

    <section class="filter-panel">
      <el-form :inline="true" :model="query" class="filter-form" @submit.prevent>
        <el-form-item label="数据库">
          <el-select v-model="query.database" class="filter-control" placeholder="请选择数据库">
            <el-option
              v-for="item in databaseOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="表名">
          <el-input v-model="query.table" class="filter-control" clearable placeholder="请输入表名" />
        </el-form-item>
        <el-form-item label="引擎">
          <el-select v-model="query.engine" class="filter-control" clearable placeholder="请选择引擎">
            <el-option
              v-for="item in engineOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="注释">
          <el-input v-model="query.comment" class="filter-control" clearable placeholder="请输入注释" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="loading" @click="loadTables">
            <IconifyIconOnline icon="ri:search-line" />
            查询
          </el-button>
          <el-button @click="resetQuery">
            <IconifyIconOnline icon="ri:restart-line" />
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </section>

    <section class="table-panel">
      <el-table :data="tables" v-loading="loading" stripe class="schema-table">
        <el-table-column prop="name" label="表名" min-width="190" show-overflow-tooltip />
        <el-table-column prop="engine" label="引擎" width="120" />
        <el-table-column prop="collation" label="字符集" min-width="180" show-overflow-tooltip />
        <el-table-column prop="rows" label="数据行数" width="120" />
        <el-table-column prop="index_size" label="索引大小" width="120">
          <template #default="{ row }">
            <el-tag type="success" effect="light">{{ row.index_size }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="comment" label="注释" min-width="190" show-overflow-tooltip>
          <template #default="{ row }">{{ row.comment || "-" }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ createdAtText(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="110" fixed="right">
          <template #default="{ row }">
            <Auth value="database:read">
              <el-button type="success" size="small" @click="openStructure(row)">
                <IconifyIconOnline icon="ri:eye-line" />
                查看
              </el-button>
            </Auth>
          </template>
        </el-table-column>
      </el-table>
    </section>

    <el-dialog
      v-model="dialogVisible"
      width="72%"
      class="schema-dialog"
      destroy-on-close
      :title="currentTable ? `表结构：${currentTable.name}` : '表结构'"
    >
      <div class="dialog-summary" v-if="currentTable">
        <span>{{ currentTable.engine }}</span>
        <span>{{ currentTable.collation || "default" }}</span>
        <span>{{ currentTable.rows }} 行</span>
        <span>{{ currentTable.index_size }}</span>
      </div>
      <el-table :data="columns" v-loading="dialogLoading" stripe max-height="520">
        <el-table-column prop="name" label="字段名称" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="field-name">{{ row.name }}</span>
            <el-tag v-if="row.primary_key" size="small" type="warning" effect="light">PK</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="type" label="类型" min-width="160" show-overflow-tooltip />
        <el-table-column label="NOT NULL" width="120">
          <template #default="{ row }">
            <el-tag :type="row.not_null ? 'primary' : 'info'" effect="light">
              {{ row.not_null ? "是" : "否" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="default" label="默认值" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">{{ row.default || "-" }}</template>
        </el-table-column>
        <el-table-column prop="comment" label="注释" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">{{ row.comment || "-" }}</template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<style scoped lang="scss">
.database-page {
  padding: 24px;
  display: grid;
  gap: 16px;
}

.database-head,
.filter-panel,
.table-panel {
  border: 1px solid var(--app-border);
  border-radius: 8px;
  background: var(--app-surface);
}

.database-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  padding: 20px 22px;
}

.eyebrow {
  margin: 0 0 6px;
  color: var(--app-cyan);
  font-family: "Menlo", "Consolas", monospace;
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
}

.database-head h2 {
  margin: 0;
  color: var(--app-text);
  font-size: 22px;
  font-weight: 760;
}

.database-head p {
  margin: 8px 0 0;
  color: var(--app-text-secondary);
}

.head-metrics {
  display: grid;
  grid-template-columns: repeat(3, minmax(110px, auto));
  gap: 10px;
}

.head-metrics div {
  padding: 12px 14px;
  border: 1px solid var(--app-border-soft);
  border-radius: 8px;
  background: var(--app-surface-soft);
}

.head-metrics span {
  display: block;
  color: var(--app-text-muted);
  font-size: 12px;
}

.head-metrics strong {
  display: block;
  margin-top: 5px;
  color: var(--app-text);
  font-size: 18px;
}

.filter-panel {
  padding: 14px 18px 0;
}

.filter-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0 8px;
}

.filter-control {
  width: 220px;
}

.table-panel {
  overflow: hidden;
}

.schema-table {
  width: 100%;
}

:deep(.el-button .iconify) {
  margin-right: 4px;
}

.dialog-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.dialog-summary span {
  padding: 5px 9px;
  color: var(--app-text-secondary);
  border: 1px solid var(--app-border);
  border-radius: 6px;
  background: var(--app-surface-soft);
}

.field-name {
  margin-right: 8px;
  color: var(--app-text);
  font-weight: 700;
}

@media (max-width: 980px) {
  .database-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .head-metrics {
    width: 100%;
    grid-template-columns: 1fr;
  }

  .filter-control {
    width: 100%;
  }

  :deep(.el-form-item) {
    width: 100%;
    margin-right: 0;
  }

  :deep(.schema-dialog) {
    width: calc(100vw - 24px) !important;
  }
}
</style>
