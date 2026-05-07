<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">余额</p>
        <p class="stat-value">
          {{ formatCurrency(session.user?.balance || 0) }}
        </p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">订单数</p>
        <p class="stat-value">{{ orders.length }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">已支付</p>
        <p class="stat-value">{{ paidOrders }}</p>
      </ElCard>
    </div>

    <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon />

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <CreditCard :size="16" />
            <span>发起充值</span>
          </div>
        </template>
        <ElForm label-position="top">
          <ElFormItem label="金额">
            <ElInput v-model.number="form.amount" type="number" placeholder="单位：1e-4 元" />
          </ElFormItem>
          <ElFormItem label="订单标题">
            <ElInput v-model="form.subject" placeholder="如 Balance Recharge" />
          </ElFormItem>
          <ElButton type="primary" :loading="submitting" @click="createOrder">创建订单</ElButton>
        </ElForm>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <ReceiptText :size="16" />
            <span>支付回执</span>
          </div>
        </template>
        <div class="pre-box">{{ prettyJSON(result) }}</div>
        <p class="helper-copy" style="margin: 12px 0 0; line-height: 1.8">一期仅接入 `gopay`。这里展示后端返回的支付参数，订单状态由回调更新，用户可在下方订单列表查看结果。</p>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <span>我的订单</span>
      </template>
      <ElTable :data="orders" empty-text="暂无订单">
        <ElTableColumn prop="out_trade_no" label="商户单号" min-width="180" />
        <ElTableColumn label="金额" min-width="120">
          <template #default="{ row }"> {{ formatCurrency(row.amount) }} 元 </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="120">
          <template #default="{ row }">
            <ElTag :type="isPaidStatus(row.status) ? 'success' : 'warning'">{{ row.status }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="创建时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.created_at_ms) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="110">
          <template #default="{ row }">
            <ElLink type="primary" @click="goDetail(row.id)">详情</ElLink>
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { CreditCard, ReceiptText } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElLink, ElTable, ElTableColumn, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import { session } from "@/store/session";
import type { PaymentCreateResponse, PaymentOrder } from "@/api/types";
import { formatCurrency, formatTime, isPaidStatus, prettyJSON } from "@/utils";

const router = useRouter();
const result = ref<PaymentCreateResponse | null>(null);
const orders = ref<PaymentOrder[]>([]);
const error = ref("");
const submitting = ref(false);
const form = reactive({
  amount: 10000,
  subject: "Balance Recharge",
});

const paidOrders = computed(() => orders.value.filter((item) => isPaidStatus(item.status)).length);

async function load() {
  orders.value = await userAPI.orders();
}

async function createOrder() {
  error.value = "";
  submitting.value = true;
  try {
    result.value = await userAPI.createPayment({
      amount: Number(form.amount || 0),
      subject: form.subject,
    });
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "create failed";
  } finally {
    submitting.value = false;
  }
}

function goDetail(id: number) {
  void router.push(`/user/orders/${id}`);
}

onMounted(() => {
  void load();
});
</script>
