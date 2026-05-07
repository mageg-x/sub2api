<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">订单总数</p>
        <p class="stat-value">{{ summary.total }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">已支付</p>
        <p class="stat-value">{{ summary.paid }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">累计金额</p>
        <p class="stat-value">{{ formatCurrency(summary.amount) }}</p>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <span>退款处理</span>
      </template>
      <ElAlert v-if="refundMessage" :title="refundMessage" type="success" :closable="false" show-icon style="margin-bottom: 16px" />
      <ElAlert v-if="refundError" :title="refundError" type="error" :closable="false" show-icon style="margin-bottom: 16px" />
      <ElForm label-position="top">
        <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
          <ElFormItem label="商户单号">
            <ElInput v-model="refundForm.out_trade_no" placeholder="输入 out_trade_no" />
          </ElFormItem>
          <ElFormItem label="退款金额">
            <ElInput v-model.number="refundForm.amount" type="number" placeholder="单位：1e-4 元" />
          </ElFormItem>
        </div>
        <ElButton type="primary" :loading="refunding" @click="refund">执行退款</ElButton>
      </ElForm>
    </ElCard>

    <DataTable
      title="支付订单"
      :columns="[
        { key: 'out_trade_no', label: '商户单号', minWidth: 180 },
        { key: 'user_id', label: '用户', width: 90 },
        { key: 'provider', label: '渠道', width: 110 },
        { key: 'amount', label: '金额', minWidth: 120 },
        { key: 'credited_amount', label: '入账金额', minWidth: 120 },
        { key: 'status', label: '状态', width: 120 },
        { key: 'created_at_ms', label: '创建时间', minWidth: 180 },
      ]"
      :rows="orders as unknown as Array<Record<string, unknown>>"
    >
      <template #amount="{ row }"> {{ formatCurrency(Number(row.amount || 0)) }} 元 </template>
      <template #credited_amount="{ row }"> {{ formatCurrency(Number(row.credited_amount || 0)) }} 元 </template>
      <template #status="{ row }">
        <ElTag :type="isPaidStatus(row.status) ? 'success' : 'warning'">{{ row.status }}</ElTag>
      </template>
      <template #created_at_ms="{ row }">
        {{ formatTime(Number(row.created_at_ms || 0)) }}
      </template>
    </DataTable>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CreditCard } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTag } from "element-plus";
import DataTable from "@/components/DataTable.vue";
import { adminAPI } from "@/api/admin";
import type { PaymentOrder } from "@/api/types";
import { formatCurrency, formatTime, isPaidStatus } from "@/utils";

const orders = ref<PaymentOrder[]>([]);
const refunding = ref(false);
const refundMessage = ref("");
const refundError = ref("");
const refundForm = reactive({
  out_trade_no: "",
  amount: 0,
});

const summary = computed(() => ({
  total: orders.value.length,
  paid: orders.value.filter((item) => isPaidStatus(item.status)).length,
  amount: orders.value.reduce((sum, item) => sum + item.amount, 0),
}));

async function load() {
  orders.value = await adminAPI.orders();
}

async function refund() {
  refundMessage.value = "";
  refundError.value = "";
  refunding.value = true;
  try {
    await adminAPI.refundOrder({
      out_trade_no: refundForm.out_trade_no,
      amount: Number(refundForm.amount || 0),
    });
    refundMessage.value = "退款已提交";
    refundForm.out_trade_no = "";
    refundForm.amount = 0;
    await load();
  } catch (err) {
    refundError.value = err instanceof Error ? err.message : "refund failed";
  } finally {
    refunding.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
