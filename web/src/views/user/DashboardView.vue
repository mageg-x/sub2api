<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">账户余额</p>
        <p class="stat-value">
          {{ formatCurrency(session.user?.balance || 0) }}
        </p>
        <p class="helper-copy" style="margin: 8px 0 0">单位：元</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">最近请求</p>
        <p class="stat-value">{{ usage.length }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">最近加载的调用记录</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">订单数量</p>
        <p class="stat-value">{{ orders.length }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">充值与回调状态追踪</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">可用角色</p>
        <p class="stat-value">{{ session.user?.role || "-" }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">当前登录身份</p>
      </ElCard>
    </div>

    <div style="display: grid; grid-template-columns: 1.1fr 0.9fr; gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <ReceiptText :size="16" />
            <span>最近调用</span>
          </div>
        </template>
        <ElEmpty v-if="usage.length === 0" description="暂无调用记录" />
        <div v-else style="display: grid; gap: 12px">
          <div v-for="item in usage.slice(0, 5)" :key="item.id" style="padding: 14px 16px; border: 1px solid var(--line); border-radius: 16px; background: rgba(255, 255, 255, 0.5)">
            <div style="display: flex; align-items: center; justify-content: space-between; gap: 12px">
              <strong>{{ item.model }}</strong>
              <ElTag size="small">{{ item.provider }}</ElTag>
            </div>
            <p class="helper-copy" style="margin: 8px 0 0">
              {{ item.endpoint }}
            </p>
            <p class="helper-copy" style="margin: 8px 0 0">输入 {{ item.input_tokens }} / 输出 {{ item.output_tokens }} / 花费 {{ formatCurrency(item.cost) }} 元</p>
          </div>
        </div>
      </ElCard>

      <div style="display: grid; gap: 18px">
        <ElCard shadow="never">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <WalletCards :size="16" />
              <span>最近订单</span>
            </div>
          </template>
          <ElEmpty v-if="!latestOrder" description="暂无订单" />
          <div v-else>
            <p class="stat-label">商户单号</p>
            <div class="mono">{{ latestOrder.out_trade_no }}</div>
            <p class="helper-copy" style="margin: 10px 0 0">
              {{ formatCurrency(latestOrder.amount) }} 元 ·
              {{ latestOrder.status }}
            </p>
            <p class="helper-copy" style="margin: 8px 0 0">
              {{ formatTime(latestOrder.created_at_ms) }}
            </p>
          </div>
        </ElCard>

        <ElCard shadow="never">
          <template #header>
            <div style="display: flex; align-items: center; gap: 8px">
              <Bell :size="16" />
              <span>闭环说明</span>
            </div>
          </template>
          <p class="helper-copy" style="margin: 0; line-height: 1.8">当前用户端已经覆盖登录、公告、兑换码、创建 API Key、查看用量、发起 gopay 订单、查看订单详情与个人资料维护。</p>
        </ElCard>
      </div>
    </div>
  </div>
</template>


<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Bell, KeyRound, ReceiptText, WalletCards } from "lucide-vue-next";
import { ElCard, ElEmpty, ElTag } from "element-plus";
import { session } from "@/store/session";
import { userAPI } from "@/api/user";
import type { PaymentOrder, UsageLog } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const usage = ref<UsageLog[]>([]);
const orders = ref<PaymentOrder[]>([]);

const latestOrder = computed(() => orders.value[0] || null);

async function load() {
  usage.value = await userAPI.usage(8);
  orders.value = await userAPI.orders();
}

onMounted(() => {
  void load();
});
</script>

