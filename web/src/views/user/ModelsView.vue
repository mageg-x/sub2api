<template>
  <div class="models-page">
    <div v-if="loadError" class="surface-card error-banner">
      <el-alert :title="loadError" type="error" :closable="false" show-icon />
    </div>
    <div class="models-layout">
      <aside class="provider-sidebar">
        <div class="provider-list">
          <button v-for="(provider, idx) in providers" :key="provider.key" class="provider-card" :class="{ active: selectedKey === provider.key, [`color-${idx % 6}`]: true }" @click="selectedKey = provider.key">
            <div class="provider-card-header">
              <span class="provider-name">{{ provider.name }}</span>
              <span class="rate-badge">{{ t('models.billingRate') }}: {{ provider.multiplier }}x</span>
            </div>
            <div class="provider-path">
              <FileText :size="14" />
              <code>{{ provider.key }}</code>
            </div>
          </button>
        </div>
      </aside>

      <main class="model-panel">
        <div class="panel-header">
          <span class="exchange-hint">{{ t('models.stationExchangeRate') }}</span>
        </div>

        <div class="model-table-wrap">
          <table class="price-table" v-if="currentModels.length">
            <thead>
              <tr>
                <th>{{ t('models.model') }}</th>
                <th>{{ t('models.inputPrice') }}</th>
                <th>{{ t('models.outputPrice') }}</th>
                <th>{{ t('models.cacheCreatePrice') }}</th>
                <th>{{ t('models.cacheReadPrice') }}</th>
                <th>{{ t('models.status') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="model in currentModels" :key="model.model">
                <td class="cell-model">{{ model.model }}</td>
                <td>
                  <span class="price-tag input">${{ model.input_price.toFixed(2) }}/M</span>
                </td>
                <td>
                  <span class="price-tag output">${{ model.output_price.toFixed(2) }}/M</span>
                </td>
                <td>
                  <span class="price-tag cache-create">${{ model.cache_create_price.toFixed(2) }}/M</span>
                </td>
                <td>
                  <span class="price-tag cache-read">${{ model.cache_read_price.toFixed(2) }}/M</span>
                </td>
                <td><span class="status-badge" :class="isActiveStatus(model.status) ? 'active' : ''">{{ t('models.available') }}</span></td>
              </tr>
            </tbody>
          </table>
          <div v-else class="empty-hint">{{ t('models.selectProvider') }}</div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { FileText } from "lucide-vue-next";
import { ElAlert } from "element-plus";
import { userAPI } from "@/api/user";
import type { ModelCatalogChannel } from "@/api/types";
import { isActiveStatus } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const catalog = ref<ModelCatalogChannel[]>([]);
const selectedKey = ref("");
const loadError = ref("");

const providers = computed(() => catalog.value);

const currentModels = computed(() => {
  if (!selectedKey.value) return [];
  const p = catalog.value.find((c) => c.key === selectedKey.value);
  return p?.models || [];
});

onMounted(async () => {
  try {
    catalog.value = await userAPI.modelCatalog();
    if (catalog.value.length) {
      selectedKey.value = catalog.value[0].key;
    }
  } catch (err) {
    loadError.value = err instanceof Error ? err.message : t('models.loadingFailed');
  }
});
</script>

<style scoped>
.models-page {
  height: calc(100vh - var(--header-height, 64px));
}

.models-layout {
  display: flex;
  gap: 0;
  height: 100%;
  border-radius: var(--radius-2xl);
  overflow: hidden;
  border: 1px solid var(--border-default);
  background: var(--bg-raised);
  box-shadow: var(--shadow-sm);
}

.provider-sidebar {
  width: 300px;
  min-width: 300px;
  border-right: 1px solid var(--border-default);
  background: var(--bg-subtle);
  overflow-y: auto;
  flex-shrink: 0;
}

.provider-sidebar::-webkit-scrollbar {
  width: 4px;
}

.provider-sidebar::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, hsl(234, 60%, 78%), hsl(262, 60%, 74%));
  border-radius: 10px;
}

.provider-list {
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.provider-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px 18px;
  border-radius: var(--radius-xl);
  border: 1.5px solid transparent;
  background: var(--bg-raised);
  cursor: pointer;
  text-align: left;
  transition: all var(--transition-fast);
  position: relative;
  overflow: hidden;
}

.provider-card:hover {
  border-color: var(--border-focus);
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.provider-card.active {
  border-color: var(--primary-color);
  box-shadow:
    0 0 0 3px var(--primary-glow),
    var(--shadow-md);
}

.provider-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.provider-name {
  font-size: 16px;
  font-weight: 800;
  color: var(--text-primary);
  letter-spacing: -0.01em;
}

.rate-badge {
  font-size: 11.5px;
  font-weight: 700;
  padding: 3px 9px;
  border-radius: var(--radius-full);
  white-space: nowrap;
  flex-shrink: 0;
}

.provider-path {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12.5px;
  color: var(--text-muted);
  background: var(--bg-subtle);
  padding: 5px 10px;
  border-radius: var(--radius-md);
  width: fit-content;
}

.provider-path code {
  font-family: var(--font-mono);
  color: var(--text-secondary);
}

.color-0 .rate-badge {
  background: #fef2f2;
  color: #dc2626;
}
.color-0.active {
  background: linear-gradient(135deg, #fff1f2, #ffe4e6);
}

.color-1 .rate-badge {
  background: #fefce8;
  color: #ca8a04;
}
.color-1.active {
  background: linear-gradient(135deg, #fefce8, #fef9c3);
}

.color-2 .rate-badge {
  background: #f0fdf4;
  color: #16a34a;
}
.color-2.active {
  background: linear-gradient(135deg, #f0fdf4, #dcfce7);
}

.color-3 .rate-badge {
  background: #eff6ff;
  color: #2563eb;
}
.color-3.active {
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
}

.color-4 .rate-badge {
  background: #fdf4ff;
  color: #c026d3;
}
.color-4.active {
  background: linear-gradient(135deg, #fdf4ff, #fae8ff);
}

.color-5 .rate-badge {
  background: #fff7ed;
  color: #ea580c;
}
.color-5.active {
  background: linear-gradient(135deg, #fff7ed, #ffedd5);
}

.model-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: var(--bg-raised);
}

.panel-header {
  padding: 14px 24px;
  border-bottom: 1px solid var(--border-default);
  background: linear-gradient(180deg, var(--bg-soft), var(--bg-subtle));
  flex-shrink: 0;
}

.exchange-hint {
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 500;
}

.model-table-wrap {
  flex: 1;
  overflow: auto;
  padding: 0;
}

.price-table {
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  table-layout: fixed;
}

.price-table th {
  position: sticky;
  top: 0;
  z-index: 2;
  background: var(--bg-subtle);
  font-size: 12.5px;
  font-weight: 700;
  color: var(--text-muted);
  letter-spacing: 0.02em;
  padding: 12px 16px;
  text-align: center;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-default);
}

.price-table td {
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-subtle);
  text-align: center;
  vertical-align: middle;
  font-size: 13.5px;
  transition: background var(--transition-fast);
}

.price-table tbody tr:hover td {
  background: var(--primary-lighter);
}

.price-table tbody tr:last-child td {
  border-bottom: none;
}

.cell-model {
  text-align: left !important;
  font-weight: 600;
  color: var(--text-primary);
  white-space: normal !important;
  word-break: break-word;
}

.price-tag {
  display: inline-block;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  font-size: 12.5px;
  font-weight: 700;
  white-space: nowrap;
  letter-spacing: -0.01em;
}

.price-tag.input {
  background: #ecfdf5;
  color: #059669;
}

.price-tag.output {
  background: #fdf2f8;
  color: #db2777;
}

.price-tag.cache-create {
  background: #fffbeb;
  color: #d97706;
}

.price-tag.cache-read {
  background: #fef2f2;
  color: #dc2626;
  opacity: 0.75;
}

.status-badge {
  display: inline-block;
  padding: 3px 12px;
  border-radius: var(--radius-full);
  font-size: 11.5px;
  font-weight: 700;
  background: var(--border-light);
  color: var(--text-muted);
}

.status-badge.active {
  background: #ecfdf5;
  color: #059669;
}

.empty-hint {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--text-muted);
  font-size: 14px;
}

@media (max-width: 900px) {
  .models-layout {
    flex-direction: column;
    height: auto;
  }

  .provider-sidebar {
    width: 100%;
    min-width: 0;
    max-height: 40vh;
    border-right: none;
    border-bottom: 1px solid var(--border-default);
  }

  .provider-list {
    flex-direction: row;
    flex-wrap: wrap;
    overflow-y: visible;
  }

  .provider-card {
    min-width: 200px;
    flex: 1;
  }

  .model-table-wrap {
    max-height: 50vh;
  }
}
</style>
