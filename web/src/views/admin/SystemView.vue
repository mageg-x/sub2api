<template>
  <div>
    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Activity :size="20" />
          {{ t('adminSystem.systemMetrics') }}
        </h3>
        <span class="stat-count">{{ statCount }} {{ t('adminSystem.items') }}</span>
      </div>
      <div class="card-body">
        <div class="stats-grid">
          <div v-for="(value, key) in displayStats" :key="String(key)" class="stat-item">
            <div class="stat-item-label">{{ formatKey(String(key)) }}</div>
            <div class="stat-item-value">{{ value }}</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Activity } from "lucide-vue-next";
import { adminAPI } from "@/api/admin";
import { formatTime } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const stats = ref<Record<string, unknown>>({});

const statCount = computed(() => Object.keys(stats.value).length);

const displayStats = computed(() => {
  const entries = Object.entries(stats.value).filter(([key]) => key !== "TIMESTAMP_MS");
  return Object.fromEntries(
    entries.map(([key, value]) => {
      if (typeof value === "number" && value > 1e12) {
        return [key, formatTime(value)];
      }
      return [key, value];
    }),
  );
});

function formatKey(key: string): string {
  return key.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

async function load() {
  try {
    stats.value = await adminAPI.stats();
  } catch {
    stats.value = {};
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.stat-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 12px;
}

.stat-item {
  padding: 14px;
  background: var(--border-light);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
  overflow: hidden;
}

.stat-item:hover {
  background: var(--border-subtle);
}

.stat-item-label {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stat-item-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.02em;
  word-break: break-all;
  line-height: 1.3;
}
</style>
