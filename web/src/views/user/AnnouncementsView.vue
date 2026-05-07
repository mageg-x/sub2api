<template>
  <div class="page-container">
    <div class="surface-card announcements-section">
      <div class="card-body empty-state">
        <Bell :size="42" />
        <h3 class="empty-title">公告已移至顶部</h3>
        <p class="empty-description">请点击右上角“公告”查看最新通知。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bell } from "lucide-vue-next";
</script>

<style scoped>
.page-container {
  animation: fadeIn 0.4s ease-out;
}

.announcements-section {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 260px;
  flex-direction: column;
  gap: 12px;
  text-align: center;
  color: var(--text-secondary);
}

.empty-title {
  margin: 0;
  font-size: 20px;
  color: var(--text-primary);
}

.empty-description {
  margin: 0;
  line-height: 1.7;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
