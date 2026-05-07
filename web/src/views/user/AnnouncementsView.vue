<template>
  <div class="page-container">
    <div class="surface-card announcements-section">
      <div class="card-body empty-state">
        <Bell :size="42" />
        <h3 class="empty-title">{{ t('announcements.announcementMoved') }}</h3>
        <p class="empty-description">{{ t('announcements.clickToViewAnnouncement') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Bell } from "lucide-vue-next";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
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
