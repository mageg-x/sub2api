<template>
  <div class="page-container">
    <div class="content-grid">
      <div v-if="error" class="surface-card error-banner">
        <el-alert :title="error" type="error" :closable="false" show-icon />
      </div>

      <div class="surface-card create-section">
        <div class="card-header">
          <h3 class="card-title">
            <PlusCircle :size="20" />
            发布新公告
          </h3>
        </div>
        <div class="card-body">
          <el-form label-position="top" class="create-form">
            <el-form-item label="公告标题" class="form-item">
              <el-input v-model="form.title" placeholder="请输入公告标题" size="large" />
            </el-form-item>
            <el-form-item label="公告内容" class="form-item">
              <el-input v-model="form.content" type="textarea" :rows="4" placeholder="请输入公告内容" size="large" />
            </el-form-item>
            <el-form-item label="状态" class="form-item">
              <el-radio-group v-model="form.status">
                <el-radio-button value="active">立即发布</el-radio-button>
                <el-radio-button value="draft">保存草稿</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-button type="primary" size="large" :loading="submitting" @click="create" class="submit-button">
              <Send :size="18" />
              发布公告
            </el-button>
          </el-form>
        </div>
      </div>

      <div class="surface-card announcements-section">
        <div class="card-header">
          <h3 class="card-title">
            <Bell :size="20" />
            公告列表
          </h3>
          <span class="announcement-count">{{ items.length }} 条公告</span>
        </div>
        <div class="card-body">
          <el-table :data="items" empty-text="暂无公告" class="modern-table" :stripe="true">
            <el-table-column prop="title" label="标题" width="140">
              <template #default="{ row }">
                <div class="title-cell">
                  <Bell :size="14" />
                  <span>{{ row.title }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="content" label="内容" min-width="200">
              <template #default="{ row }">
                <span class="content-text">{{ row.content }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                  {{ row.status === "active" ? "已发布" : "草稿" }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="发布时间" width="140">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.published_at_ms) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Bell, PlusCircle, Send } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput, ElRadioButton, ElRadioGroup, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Announcement } from "@/api/types";
import { formatTime } from "@/utils";

const items = ref<Announcement[]>([]);
const submitting = ref(false);
const error = ref("");
const form = ref({
  title: "",
  content: "",
  status: "active",
});

async function load() {
  try {
    const data = await adminAPI.dashboard();
    items.value = data.announcements || [];
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载失败";
  }
}

async function create() {
  if (!form.value.title || !form.value.content) {
    error.value = "请填写标题和内容";
    return;
  }

  error.value = "";
  submitting.value = true;
  try {
    await adminAPI.createAnnouncement({
      title: form.value.title,
      content: form.value.content,
      status: form.value.status,
    });
    form.value = { title: "", content: "", status: "active" };
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "创建失败";
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.page-container {
  animation: fadeIn 0.4s ease-out;
}

.content-grid {
  display: grid;
  gap: 24px;
}

.error-banner {
  padding: 0;
}

.create-section,
.announcements-section {
  overflow: visible;
}

.create-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-item {
  margin-bottom: 16px;
}

.submit-button {
  align-self: flex-start;
  min-width: 160px;
}

.announcement-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.modern-table {
  overflow: visible;
}

.title-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: var(--text-primary);
}

.title-cell svg {
  color: var(--primary-color);
}

.content-text {
  font-size: 13px;
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  max-width: 300px;
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
