<script setup lang="ts">
defineOptions({
  name: "RePagination"
});

const props = withDefaults(
  defineProps<{
    page: number;
    pageSize: number;
    total: number;
    pageSizes?: number[];
    disabled?: boolean;
  }>(),
  {
    pageSizes: () => [10, 20, 50, 100],
    disabled: false
  }
);

const emit = defineEmits<{
  (event: "update:page", value: number): void;
  (event: "update:pageSize", value: number): void;
  (event: "change"): void;
}>();

const handleSizeChange = (value: number) => {
  emit("update:pageSize", value);
  emit("update:page", 1);
  emit("change");
};

const handleCurrentChange = (value: number) => {
  emit("update:page", value);
  emit("change");
};

const summaryPrefix = "\u5171";
const summarySuffix = "\u6761\u8bb0\u5f55";
</script>

<template>
  <div v-if="props.total > 0" class="re-pagination">
    <div class="re-pagination__summary">
      {{ summaryPrefix }} <strong>{{ props.total }}</strong> {{ summarySuffix }}
    </div>
    <el-pagination
      background
      :current-page="props.page"
      :page-size="props.pageSize"
      :page-sizes="props.pageSizes"
      :total="props.total"
      :disabled="props.disabled"
      layout="sizes, prev, pager, next, jumper"
      @size-change="handleSizeChange"
      @current-change="handleCurrentChange"
    />
  </div>
</template>

<style lang="scss" scoped>
.re-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: var(--app-surface);
  border-top: 1px solid var(--app-border-soft);
}

.re-pagination__summary {
  flex: 0 0 auto;
  font-size: 13px;
  color: var(--app-text-muted);

  strong {
    color: var(--app-primary);
    font-weight: 700;
  }
}

:deep(.el-pagination) {
  --el-pagination-button-bg-color: #f6f8fb;
  --el-pagination-hover-color: var(--app-primary);

  justify-content: flex-end;
  margin-left: auto;
}

:deep(.el-pagination.is-background .btn-next),
:deep(.el-pagination.is-background .btn-prev),
:deep(.el-pagination.is-background .el-pager li) {
  border: 1px solid var(--app-border);
  border-radius: 6px;
}

:deep(.el-pagination.is-background .el-pager li.is-active) {
  border-color: var(--app-primary);
  background: var(--app-primary);
}

@media (width <= 768px) {
  .re-pagination {
    align-items: flex-start;
    flex-direction: column;
  }

  :deep(.el-pagination) {
    justify-content: flex-start;
    margin-left: 0;
  }
}
</style>
