<template>
  <el-card shadow="never">
    <template v-if="title" #header>
      <div style="display: flex; align-items: center; justify-content: space-between; gap: 12px">
        <span>{{ title }}</span>
        <el-tag type="info" effect="plain">{{ rows.length }} 条</el-tag>
      </div>
    </template>
    <el-table :data="rows" empty-text="暂无数据" style="width: 100%">
      <el-table-column v-for="column in columns" :key="column.key" :prop="column.key" :label="column.label" :width="column.width" :min-width="column.minWidth || 120" show-overflow-tooltip>
        <template #default="{ row }">
          <slot :name="column.key" :row="row">
            {{ row[column.key] }}
          </slot>
        </template>
      </el-table-column>
    </el-table>
  </el-card>
</template>

<script setup lang="ts">
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


