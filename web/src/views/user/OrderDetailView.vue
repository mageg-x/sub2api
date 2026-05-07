<template>
  <ElCard shadow="never">
    <template #header>
      <div style="display: flex; align-items: center; gap: 8px">
        <ReceiptText :size="16" />
        <span>订单详情</span>
      </div>
    </template>
    <ElDescriptions v-if="order" :column="2" border>
      <ElDescriptionsItem label="商户单号">{{ order.out_trade_no }}</ElDescriptionsItem>
      <ElDescriptionsItem label="状态">{{ order.status }}</ElDescriptionsItem>
      <ElDescriptionsItem label="金额">{{ formatCurrency(order.amount) }} 元</ElDescriptionsItem>
      <ElDescriptionsItem label="入账金额">{{ formatCurrency(order.credited_amount) }}</ElDescriptionsItem>
      <ElDescriptionsItem label="支付渠道">{{ order.provider }}</ElDescriptionsItem>
      <ElDescriptionsItem label="创建时间">{{ formatTime(order.created_at_ms) }}</ElDescriptionsItem>
    </ElDescriptions>
  </ElCard>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { ReceiptText } from "lucide-vue-next";
import { ElCard, ElDescriptions, ElDescriptionsItem } from "element-plus";
import { userAPI } from "@/api/user";
import type { PaymentOrder } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const route = useRoute();
const order = ref<PaymentOrder | null>(null);

async function load() {
  order.value = await userAPI.orderByID(Number(route.params.id));
}

onMounted(() => {
  void load();
});
</script>


