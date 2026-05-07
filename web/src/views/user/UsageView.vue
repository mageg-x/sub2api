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
        <p class="stat-value">{{ summary.requests }}</p>
        <p class="stat-helper">API 请求总计</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <ArrowDownToLine :size="24" />
          </div>
        </div>
        <p class="stat-label">输入 Tokens</p>
        <p class="stat-value">{{ summary.input.toLocaleString() }}</p>
        <p class="stat-helper">Prompt tokens</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <ArrowUpFromLine :size="24" />
          </div>
        </div>
        <p class="stat-label">输出 Tokens</p>
        <p class="stat-value">{{ summary.output.toLocaleString() }}</p>
        <p class="stat-helper">Completion tokens</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <ReceiptText :size="24" />
          </div>
        </div>
        <p class="stat-label">花费</p>
        <p class="stat-value">{{ formatCurrency(summary.cost) }}</p>
        <p class="stat-helper">累计消费金额</p>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <ListOrdered :size="20" />
          我的用量
        </h3>
      </div>
      <div class="card-body">
        <el-table :data="usage" empty-text="暂无记录" class="modern-table" :stripe="true">
          <el-table-column label="时间" min-width="170">
            <template #default="{ row }">
              {{ formatTime(Number(row.created_at_ms || 0)) }}
            </template>
          </el-table-column>
          <el-table-column prop="provider" label="Provider" width="110">
            <template #default="{ row }">
              <el-tag size="small">{{ row.provider }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="model" label="模型" min-width="160" />
          <el-table-column prop="endpoint" label="接口" min-width="180" class-name="mono-cell" />
          <el-table-column prop="input_tokens" label="输入" width="90" align="right" />
          <el-table-column prop="output_tokens" label="输出" width="90" align="right" />
          <el-table-column label="费用" width="100" align="right">
            <template #default="{ row }">
              <span class="cost-text">{{ formatCurrency(Number(row.cost || 0)) }} 元</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ArrowDownToLine, ArrowUpFromLine, Bolt, ListOrdered, ReceiptText } from "lucide-vue-next";
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

<style scoped>
.cost-text {
  color: var(--warning-color);
  font-weight: 600;
}
</style>