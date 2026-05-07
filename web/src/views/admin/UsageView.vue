<template>
  <div>
    <div class="card-grid">
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <Bolt :size="24" />
          </div>
        </div>
        <p class="stat-label">调用次数</p>
        <p class="stat-value">{{ summary.requests.toLocaleString() }}</p>
        <p class="stat-helper">API 请求总计</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon input">
            <ArrowDownToLine :size="24" />
          </div>
        </div>
        <p class="stat-label">输入 Tokens</p>
        <p class="stat-value">{{ summary.input.toLocaleString() }}</p>
        <p class="stat-helper">Prompt tokens</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon output">
            <ArrowUpFromLine :size="24" />
          </div>
        </div>
        <p class="stat-label">输出 Tokens</p>
        <p class="stat-value">{{ summary.output.toLocaleString() }}</p>
        <p class="stat-helper">Completion tokens</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon cost">
            <ReceiptText :size="24" />
          </div>
        </div>
        <p class="stat-label">累计成本</p>
        <p class="stat-value">{{ formatCurrency(summary.cost) }}</p>
        <p class="stat-helper">元</p>
      </div>
    </div>

    <DataTable
      title="全站用量日志"
      :columns="[
        { key: 'created_at_ms', label: '时间', minWidth: 170 },
        { key: 'user_id', label: '用户', width: 80 },
        { key: 'provider', label: 'Provider', width: 110 },
        { key: 'model', label: '模型', minWidth: 160 },
        { key: 'endpoint', label: '接口', minWidth: 160 },
        { key: 'input_tokens', label: '输入', width: 90 },
        { key: 'output_tokens', label: '输出', width: 90 },
        { key: 'cost', label: '费用', minWidth: 110 },
      ]"
      :rows="usage as unknown as Array<Record<string, unknown>>"
    >
      <template #created_at_ms="{ row }">
        {{ formatTime(Number(row.created_at_ms || 0)) }}
      </template>
      <template #cost="{ row }"> {{ formatCurrency(Number(row.cost || 0)) }} 元 </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ArrowDownToLine, ArrowUpFromLine, Bolt, ReceiptText } from "lucide-vue-next";
import DataTable from "@/components/DataTable.vue";
import { adminAPI } from "@/api/admin";
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
  usage.value = await adminAPI.usage();
}

onMounted(() => {
  void load();
});
</script>