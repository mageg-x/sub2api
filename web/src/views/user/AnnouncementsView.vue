<template>
  <div class="page-container">
    <div class="content-grid">
      <div class="surface-card announcements-section">
        <div class="card-header">
          <h3 class="card-title">
            <Bell :size="20" />
            公告列表
          </h3>
          <span class="announcement-count">{{ items.length }} 条公告</span>
        </div>
        <div class="card-body">
          <el-empty v-if="items.length === 0" description="暂无公告" class="custom-empty">
            <template #image>
              <div class="empty-illustration">
                <Bell :size="48" />
              </div>
            </template>
          </el-empty>
          <div v-else class="timeline-container">
            <el-timeline>
              <el-timeline-item 
                v-for="item in items" 
                :key="item.id"
                :timestamp="formatTime(item.published_at_ms)"
                placement="top"
                size="large"
              >
                <div class="announcement-card">
                  <div class="announcement-header">
                    <h4 class="announcement-title">{{ item.title }}</h4>
                    <el-tag :type="item.status === 'active' ? 'success' : 'info'" size="small">
                      {{ item.status === 'active' ? '进行中' : '已结束' }}
                    </el-tag>
                  </div>
                  <p class="announcement-content">{{ item.content }}</p>
                </div>
              </el-timeline-item>
            </el-timeline>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { Bell } from "lucide-vue-next";
import { ElEmpty, ElTag, ElTimeline, ElTimelineItem } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Announcement } from "@/api/types";
import { formatTime } from "@/utils";

const items = ref<Announcement[]>([]);

async function load() {
  items.value = await adminAPI.announcements();
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

.announcements-section {
  overflow: visible;
}

.announcement-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.custom-empty {
  padding: 60px 20px;
}

.empty-illustration {
  width: 100px;
  height: 100px;
  background: var(--primary-lighter);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary-color);
  margin: 0 auto;
}

.timeline-container {
  padding: 20px 0;
}

.announcement-card {
  background: var(--border-light);
  border-radius: var(--radius-xl);
  padding: 24px;
  border: 1px solid var(--border-color);
  transition: all var(--transition-normal);
}

.announcement-card:hover {
  background: white;
  box-shadow: var(--card-shadow);
  transform: translateY(-2px);
}

.announcement-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.announcement-title {
  font-size: 18px;
  font-weight: 700;
  margin: 0;
  color: var(--text-primary);
  flex: 1;
}

.announcement-content {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.8;
}

:deep(.el-timeline-item__node--large) {
  width: 16px;
  height: 16px;
  background: var(--primary-color);
  border: 3px solid var(--primary-lighter);
  box-shadow: 0 0 0 4px rgba(102, 126, 234, 0.1);
}

:deep(.el-timeline-item__tail) {
  border-left: 2px solid var(--border-color);
}

:deep(.el-timeline-item__timestamp) {
  font-size: 13px;
  color: var(--text-muted);
  font-weight: 500;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
