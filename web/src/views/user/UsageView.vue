<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">调用次数</p>
        <p class="stat-value">{{ summary.requests }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">输入 Tokens</p>
        <p class="stat-value">{{ summary.input }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">输出 Tokens</p>
        <p class="stat-value">{{ summary.output }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">花费</p>
        <p class="stat-value">{{ formatCurrency(summary.cost) }}</p>
      </ElCard>
    </div>

    <DataTable
      title="我的用量"
      :columns="[
        { key: 'created_at_ms', label: '时间', minWidth: 180 },
        { key: 'provider', label: 'Provider', width: 120 },
        { key: 'model', label: '模型', minWidth: 180 },
        { key: 'endpoint', label: '接口', minWidth: 180 },
        { key: 'input_tokens', label: '输入', width: 100 },
        { key: 'output_tokens', label: '输出', width: 100 },
        { key: 'cost', label: '费用', minWidth: 120 },
      ]"
      :rows="usage as unknown as Array<Record<string, unknown>>"
    >
      <template #created_at_ms="{ row }">
        {{ formatTime(Number(row.created_at_ms || 0)) }}
      </template>
      <template #provider="{ row }">
        <ElTag size="small">{{ row.provider }}</ElTag>
      </template>
      <template #cost="{ row }"> {{ formatCurrency(Number(row.cost || 0)) }} 元 </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElCard, ElTag } from "element-plus";
import DataTable from "@/components/DataTable.vue";
import { userAPI } from "@/api/user";
import type { UsageLog } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const usage = ref<UsageLog[]>([]);

const summary = computed(() => ({
  requests: usage.value.length,
  input: usage.value.reduce((sum, item) => sum + item.input_tokens, 0),
  output: usage.value.reduce((sum, item) => sum + item.output_tokens, 0),
  cost: usage.value.reduce((sum, item) => sum + item.cost, 0),
}));

async function load() {
  usage.value = await userAPI.usage();
}

onMounted(() => {
  void load();
});
</script>


