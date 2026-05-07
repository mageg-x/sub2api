<template>
  <div>
    <div class="card-grid" style="grid-template-columns: 1fr 1fr">
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon error">
            <Bug :size="24" />
          </div>
        </div>
        <p class="stat-label">错误总数</p>
        <p class="stat-value">{{ items.length }}</p>
        <p class="stat-helper">记录的错误条目</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon alert">
            <AlertTriangle :size="24" />
          </div>
        </div>
        <p class="stat-label">最近 24h</p>
        <p class="stat-value">{{ recentItems.length }}</p>
        <p class="stat-helper">近期错误</p>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Bug :size="20" />
          错误日志
        </h3>
        <span class="error-count">{{ items.length }} 条</span>
      </div>
      <div class="card-body">
        <el-table :data="items" empty-text="暂无错误记录" class="modern-table" :stripe="true">
          <el-table-column prop="scope" label="范围" width="100">
            <template #default="{ row }">
              <el-tag type="warning" size="small">{{ row.scope }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="错误消息" min-width="200">
            <template #default="{ row }">
              <span class="error-message">{{ row.message }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="detail" label="详情" min-width="220">
            <template #default="{ row }">
              <span class="error-detail">{{ row.detail || "-" }}</span>
            </template>
          </el-table-column>
          <el-table-column label="次数" width="70" align="center">
            <template #default="{ row }">
              <span class="count-badge">{{ row.count }}</span>
            </template>
          </el-table-column>
          <el-table-column label="最近出现" width="150">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(Number(row.last_seen_at_ms || 0)) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { AlertTriangle, Bug } from "lucide-vue-next";
import { ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import { formatTime } from "@/utils";

const items = ref<Array<Record<string, unknown>>>([]);

const recentItems = computed(() => {
  const dayAgo = Date.now() - 24 * 60 * 60 * 1000;
  return items.value.filter((item) => Number(item.last_seen_at_ms || 0) > dayAgo);
});

async function load() {
  items.value = await adminAPI.errors();
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.error-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.error-message {
  font-size: 13px;
  color: var(--danger-color);
  font-weight: 500;
}

.error-detail {
  font-size: 12px;
  color: var(--text-muted);
  font-family: var(--font-mono, monospace);
}

.count-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 24px;
  background: var(--danger-light);
  color: var(--danger-color);
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 700;
  padding: 0 8px;
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}
</style>
