<template>
  <div style="display: grid; gap: 18px">
    <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon />

    <div class="card-grid">
      <ElCard v-for="item in metricCards" :key="item.label" shadow="never" class="stat-card">
        <div style="display: flex; align-items: flex-start; justify-content: space-between; gap: 12px">
          <div>
            <p class="stat-label">{{ item.label }}</p>
            <p class="stat-value">{{ item.value }}</p>
            <p class="helper-copy" style="margin: 8px 0 0">{{ item.helper }}</p>
          </div>
          <component :is="item.icon" :size="18" style="color: var(--accent)" />
        </div>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <CircleDollarSign :size="16" />
          <span>系统指标</span>
        </div>
      </template>
      <ElEmpty v-if="!loading && statEntries.length === 0" description="暂无指标" />
      <div v-else style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 14px">
        <div v-for="[key, value] in statEntries" :key="key" style="padding: 14px 16px; border: 1px solid var(--line); border-radius: 16px; background: rgba(255, 255, 255, 0.5)">
          <div class="stat-label">{{ key }}</div>
          <div style="font-size: 24px; margin-top: 6px">{{ value }}</div>
        </div>
      </div>
    </ElCard>

    <div style="display: grid; grid-template-columns: 1.1fr 0.9fr; gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <Boxes :size="16" />
            <span>最近账户池</span>
          </div>
        </template>
        <ElTable :data="data?.accounts?.slice(0, 8) || []" empty-text="暂无账户">
          <ElTableColumn prop="provider" label="Provider" min-width="120" />
          <ElTableColumn prop="name" label="账户名" min-width="150" />
          <ElTableColumn prop="auth_type" label="认证方式" min-width="120" />
          <ElTableColumn label="状态" width="110">
            <template #default="{ row }">
              <ElTag :type="isActiveStatus(row.status) ? 'success' : 'info'">{{ row.status }}</ElTag>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <Bell :size="16" />
            <span>最近公告</span>
          </div>
        </template>
        <ElEmpty v-if="!data?.announcements?.length" description="暂无公告" />
        <div v-else style="display: grid; gap: 12px">
          <div v-for="item in data.announcements.slice(0, 4)" :key="item.id" style="padding: 14px 16px; border: 1px solid var(--line); border-radius: 16px; background: rgba(255, 255, 255, 0.5)">
            <div style="display: flex; align-items: center; justify-content: space-between; gap: 12px">
              <strong>{{ item.title }}</strong>
              <ElTag size="small">{{ item.status }}</ElTag>
            </div>
            <p class="helper-copy" style="margin: 8px 0 0">
              {{ item.content }}
            </p>
            <p class="helper-copy" style="margin: 8px 0 0">
              {{ formatTime(item.published_at_ms) }}
            </p>
          </div>
        </div>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <CreditCard :size="16" />
          <span>最近订单</span>
        </div>
      </template>
      <ElTable :data="data?.orders?.slice(0, 8) || []" empty-text="暂无订单">
        <ElTableColumn prop="out_trade_no" label="商户单号" min-width="180" />
        <ElTableColumn prop="user_id" label="用户" width="90" />
        <ElTableColumn prop="provider" label="渠道" width="110" />
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
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Bell, Boxes, CircleDollarSign, CreditCard, KeyRound, Users } from "lucide-vue-next";
import { ElAlert, ElCard, ElEmpty, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { DashboardResponse } from "@/api/types";
import { formatCurrency, formatTime, isActiveStatus, isPaidStatus } from "@/utils";

const data = ref<DashboardResponse | null>(null);
const loading = ref(false);
const error = ref("");

const metricCards = computed(() => {
  if (!data.value) return [];
  return [
    {
      label: "用户数",
      value: data.value.users.length,
      helper: "平台注册用户",
      icon: Users,
    },
    {
      label: "API Keys",
      value: data.value.api_keys.length,
      helper: "已发放访问凭据",
      icon: KeyRound,
    },
    {
      label: "上游账户",
      value: data.value.accounts.length,
      helper: "OAuth / 静态密钥混合池",
      icon: Boxes,
    },
    {
      label: "支付订单",
      value: data.value.orders.length,
      helper: "一期仅 gopay",
      icon: CreditCard,
    },
  ];
});

const statEntries = computed(() => Object.entries(data.value?.stats || {}));

async function load() {
  loading.value = true;
  error.value = "";
  try {
    data.value = await adminAPI.dashboard();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "load failed";
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

