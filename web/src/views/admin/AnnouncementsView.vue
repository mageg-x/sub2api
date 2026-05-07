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
            {{ t('adminAnnouncements.publishNewAnnouncement') }}
          </h3>
        </div>
        <div class="card-body">
          <el-form label-position="top" class="create-form">
            <el-form-item :label="t('adminAnnouncements.announcementTitle')" class="form-item">
              <el-input v-model="form.title" :placeholder="t('adminAnnouncements.pleaseInputTitle')" size="large" />
            </el-form-item>
            <el-form-item :label="t('adminAnnouncements.announcementContent')" class="form-item">
              <el-input v-model="form.content" type="textarea" :rows="4" :placeholder="t('adminAnnouncements.pleaseInputContent')" size="large" />
            </el-form-item>
            <el-form-item :label="t('adminAnnouncements.status')" class="form-item">
              <el-radio-group v-model="form.status">
                <el-radio-button value="active">{{ t('adminAnnouncements.publishNow') }}</el-radio-button>
                <el-radio-button value="draft">{{ t('adminAnnouncements.saveDraft') }}</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-button type="primary" size="large" :loading="submitting" @click="create" class="submit-button">
              <Send :size="18" />
              {{ t('adminAnnouncements.publishAnnouncement') }}
            </el-button>
          </el-form>
        </div>
      </div>

      <div class="surface-card announcements-section">
        <div class="card-header">
          <h3 class="card-title">
            <Bell :size="20" />
            {{ t('adminAnnouncements.announcementList') }}
          </h3>
          <span class="announcement-count">{{ items.length }} {{ t('adminAnnouncements.announcementCount') }}</span>
        </div>
        <div class="card-body">
          <el-table :data="items" :empty-text="t('adminAnnouncements.noAnnouncements')" class="modern-table" :stripe="true">
            <el-table-column prop="title" :label="t('adminAnnouncements.title')" width="140">
              <template #default="{ row }">
                <div class="title-cell">
                  <Bell :size="14" />
                  <span>{{ row.title }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="content" :label="t('adminAnnouncements.content')" min-width="200">
              <template #default="{ row }">
                <span class="content-text">{{ row.content }}</span>
              </template>
            </el-table-column>
            <el-table-column :label="t('adminAnnouncements.status')" width="80">
              <template #default="{ row }">
                <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                  {{ row.status === "active" ? t('adminAnnouncements.published') : t('adminAnnouncements.draft') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('adminAnnouncements.publishTime')" width="140">
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
import { useI18n } from "vue-i18n";

const { t } = useI18n();
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
    error.value = err instanceof Error ? err.message : t('adminAnnouncements.loadFailed');
  }
}

async function create() {
  if (!form.value.title || !form.value.content) {
    error.value = t('adminAnnouncements.pleaseFillTitleAndContent');
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
    error.value = err instanceof Error ? err.message : t('adminAnnouncements.createFailed');
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
