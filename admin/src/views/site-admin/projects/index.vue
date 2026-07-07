<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import {
  deleteSiteProject,
  getSiteProjects,
  saveSiteProject,
  type SiteProject
} from "@/api/admin";
import { message } from "@/utils/message";

defineOptions({ name: "SiteAdminProjects" });

type ProjectForm = Partial<SiteProject>;

const emptyForm = (): ProjectForm => ({
  name: "",
  summary: "",
  description: "",
  cover_url: "",
  demo_url: "",
  repo_url: "",
  stack_tags: "",
  role: "",
  highlights: "",
  metrics: "",
  challenge: "",
  solution: "",
  gallery_json: "",
  status: "draft",
  is_featured: false,
  sort_order: 0,
  priority: 0
});

const loading = ref(false);
const saving = ref(false);
const drawerVisible = ref(false);
const projects = ref<SiteProject[]>([]);
const total = ref(0);
const current = ref<SiteProject>();
const query = reactive({ page: 1, page_size: 12, status: "" });
const form = reactive<ProjectForm>(emptyForm());

const loadData = async () => {
  loading.value = true;
  try {
    const res = await getSiteProjects(query);
    projects.value = res.projects ?? [];
    total.value = res.total ?? 0;
  } catch {
    message("项目作品加载失败", { type: "error" });
  } finally {
    loading.value = false;
  }
};

const openCreate = () => {
  current.value = undefined;
  Object.assign(form, emptyForm());
  drawerVisible.value = true;
};

const openEdit = (row: SiteProject) => {
  current.value = row;
  Object.assign(form, row);
  drawerVisible.value = true;
};

const saveData = async () => {
  saving.value = true;
  try {
    await saveSiteProject(form, current.value?.id);
    message("项目作品已保存", { type: "success" });
    drawerVisible.value = false;
    await loadData();
  } catch {
    message("项目作品保存失败", { type: "error" });
  } finally {
    saving.value = false;
  }
};

const remove = async (row: SiteProject) => {
  try {
    await deleteSiteProject(row.id);
    message("项目作品已删除", { type: "success" });
    await loadData();
  } catch {
    message("项目作品删除失败", { type: "error" });
  }
};

const search = () => {
  query.page = 1;
  loadData();
};

const timeText = (value?: string) =>
  value ? dayjs(value).format("YYYY-MM-DD") : "-";

onMounted(loadData);
</script>

<template>
  <div class="projects-page" v-loading="loading">
    <section class="page-head">
      <div>
        <p class="eyebrow">Portfolio desk</p>
        <h2>项目作品管理</h2>
        <p>维护官网作品案例，补齐角色、亮点、指标、挑战和解决方案，便于前台更完整展示。</p>
      </div>
      <div class="head-actions">
        <el-select v-model="query.status" placeholder="状态" clearable @change="search">
          <el-option label="published" value="published" />
          <el-option label="draft" value="draft" />
        </el-select>
        <el-button @click="loadData">刷新</el-button>
        <el-button type="primary" @click="openCreate">新建项目</el-button>
      </div>
    </section>

    <section class="panel">
      <el-table :data="projects" stripe>
        <el-table-column prop="name" label="项目" min-width="180" show-overflow-tooltip />
        <el-table-column prop="summary" label="摘要" min-width="240" show-overflow-tooltip />
        <el-table-column prop="role" label="角色" width="130" show-overflow-tooltip />
        <el-table-column prop="stack_tags" label="技术栈" min-width="180" show-overflow-tooltip />
        <el-table-column prop="priority" label="优先级" width="90" />
        <el-table-column label="精选" width="80">
          <template #default="{ row }">
            <el-tag :type="row.is_featured ? 'success' : 'info'" effect="light">
              {{ row.is_featured ? "是" : "否" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="row.status === 'published' ? 'success' : 'info'" effect="light">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="发布时间" width="130">
          <template #default="{ row }">{{ timeText(row.published_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm title="确认删除这个项目？" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        v-model:current-page="query.page"
        v-model:page-size="query.page_size"
        layout="total, prev, pager, next"
        :total="total"
        @current-change="loadData"
      />
    </section>

    <el-drawer v-model="drawerVisible" :title="current ? '编辑项目' : '新建项目'" size="720px">
      <el-form label-position="top" class="project-form">
        <div class="form-grid">
          <el-form-item label="项目名称" required>
            <el-input v-model="form.name" placeholder="例如：RAG 知识库后台" />
          </el-form-item>
          <el-form-item label="承担角色">
            <el-input v-model="form.role" placeholder="全栈开发 / 产品设计" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="form.status">
              <el-option label="draft" value="draft" />
              <el-option label="published" value="published" />
            </el-select>
          </el-form-item>
          <el-form-item label="优先级">
            <el-input-number v-model="form.priority" :min="0" controls-position="right" />
          </el-form-item>
        </div>

        <el-form-item label="摘要">
          <el-input v-model="form.summary" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="详细描述">
          <el-input v-model="form.description" type="textarea" :rows="4" />
        </el-form-item>

        <div class="form-grid">
          <el-form-item label="封面地址">
            <el-input v-model="form.cover_url" />
          </el-form-item>
          <el-form-item label="技术栈">
            <el-input v-model="form.stack_tags" placeholder="Vue, Go, PostgreSQL" />
          </el-form-item>
          <el-form-item label="演示地址">
            <el-input v-model="form.demo_url" />
          </el-form-item>
          <el-form-item label="仓库地址">
            <el-input v-model="form.repo_url" />
          </el-form-item>
        </div>

        <el-form-item label="项目亮点">
          <el-input v-model="form.highlights" type="textarea" :rows="3" placeholder="每行一条亮点" />
        </el-form-item>
        <el-form-item label="结果指标">
          <el-input v-model="form.metrics" type="textarea" :rows="3" placeholder="例如：响应耗时降低 40%" />
        </el-form-item>
        <el-form-item label="挑战">
          <el-input v-model="form.challenge" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="解决方案">
          <el-input v-model="form.solution" type="textarea" :rows="4" />
        </el-form-item>
        <el-form-item label="图库 JSON">
          <el-input v-model="form.gallery_json" type="textarea" :rows="3" placeholder='["/uploads/site/demo.png"]' />
        </el-form-item>

        <div class="form-grid">
          <el-form-item label="开始时间">
            <el-date-picker v-model="form.start_date" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
          </el-form-item>
          <el-form-item label="结束时间">
            <el-date-picker v-model="form.end_date" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
          </el-form-item>
          <el-form-item label="发布时间">
            <el-date-picker v-model="form.published_at" type="datetime" value-format="YYYY-MM-DDTHH:mm:ssZ" />
          </el-form-item>
          <el-form-item label="精选展示">
            <el-switch v-model="form.is_featured" />
          </el-form-item>
        </div>
      </el-form>

      <template #footer>
        <el-button @click="drawerVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveData">保存项目</el-button>
      </template>
    </el-drawer>
  </div>
</template>

<style scoped lang="scss">
.projects-page {
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

.page-head h2 {
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

.head-actions .el-select {
  width: 140px;
}

.panel {
  padding: 18px;
}

.project-form {
  padding-right: 8px;
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.el-date-editor {
  width: 100%;
}

@media (max-width: 900px) {
  .page-head,
  .form-grid {
    display: grid;
    grid-template-columns: 1fr;
  }
}
</style>
