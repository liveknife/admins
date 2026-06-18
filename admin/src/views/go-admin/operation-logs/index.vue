<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import { getOperationLogs, type OperationLog } from "@/api/admin";
import { message } from "@/utils/message";
import RePagination from "@/components/RePagination";

defineOptions({ name: "GoAdminOperationLogs" });

const loading = ref(false);
const logs = ref<OperationLog[]>([]);
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
});

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getOperationLogs({
      page: pagination.page,
      page_size: pagination.pageSize
    });
    logs.value = res.logs ?? [];
    pagination.total = res.total ?? 0;
  } catch {
    message("操作日志加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

onMounted(loadData);
</script>

<template>
  <div class="log-page">
    <div class="page-head">
      <div>
        <h2>操作日志</h2>
        <p>记录管理员在用户管理、角色权限等模块中的关键操作。</p>
      </div>
      <el-button type="primary" @click="loadData">刷新</el-button>
    </div>

    <div class="table-panel">
      <el-table :data="logs" v-loading="loading" stripe class="log-table">
        <el-table-column prop="id" label="ID" width="88" />
        <el-table-column prop="username" label="操作人" min-width="120">
          <template #default="{ row }">{{ row.username || "系统" }}</template>
        </el-table-column>
        <el-table-column prop="action" label="操作" min-width="140" />
        <el-table-column prop="resource" label="模块" min-width="120" />
        <el-table-column prop="detail" label="详情" min-width="220" show-overflow-tooltip />
        <el-table-column prop="ip" label="IP" min-width="130" />
        <el-table-column prop="user_agent" label="客户端" min-width="220" show-overflow-tooltip />
        <el-table-column label="时间" width="180">
          <template #default="{ row }">
            {{ dayjs(row.created_at).format("YYYY-MM-DD HH:mm:ss") }}
          </template>
        </el-table-column>
      </el-table>
      <RePagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :disabled="loading"
        @change="loadData"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.log-page {
  padding: 24px;
}
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.page-head h2 {
  margin: 0;
  font-size: 20px;
}
.page-head p {
  margin: 6px 0 0;
  color: var(--app-text-secondary);
}
.table-panel {
  background: var(--app-surface);
  border: 1px solid var(--app-border);
  border-radius: 8px;
  overflow: hidden;
}
.log-table {
  width: 100%;
}
</style>
