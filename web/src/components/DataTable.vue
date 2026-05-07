<template>
  <div class="surface-card data-table-card">
    <div v-if="title" class="card-header">
      <h3 class="card-title">{{ title }}</h3>
      <span class="row-count">{{ rows.length }} {{ t('dataTable.rowCount') }}</span>
    </div>
    <div class="card-body">
      <el-table :data="rows" :empty-text="t('dataTable.noData')" class="modern-table" :stripe="true">
        <el-table-column v-for="column in columns" :key="column.key" :prop="column.key" :label="column.label" :width="column.width" :min-width="column.minWidth || 120" show-overflow-tooltip>
          <template #default="{ row }">
            <slot :name="column.key" :row="row">
              {{ row[column.key] }}
            </slot>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

const { t } = useI18n();

defineProps<{
  title?: string;
  columns: Array<{
    key: string;
    label: string;
    width?: number | string;
    minWidth?: number | string;
  }>;
  rows: Array<Record<string, unknown>>;
}>();
</script>

<style scoped>
.row-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}
</style>
