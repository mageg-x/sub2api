<template>
  <div>
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon-col blue">
          <WalletCards :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.currentBalance") }}</span>
          <span class="stat-value">{{ formatCurrency(balance) }}</span>
        </div>
        <el-button type="primary" size="small" class="stat-action" @click="router.push('/user/payment')">
          {{ t("dashboard.recharge") }}
        </el-button>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col purple">
          <TrendingDown :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.historicalConsumption") }}</span>
          <span class="stat-value">{{ formatCurrency(totalCost) }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col teal">
          <Send :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.requestCount") }}</span>
          <span class="stat-value">{{ requestCount }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="requestSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col cyan">
          <BarChart3 :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.requestsLast7Days") }}</span>
          <span class="stat-value">{{ recentUsageCount }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="recentSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col yellow">
          <Coins :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.statsQuota") }}</span>
          <span class="stat-value">{{ formatCurrency(totalCost) }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="costSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col rose">
          <Type :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.totalTokens") }}</span>
          <span class="stat-value">{{ formatNumber(totalTokens) }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="tokenSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col violet">
          <Timer :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.avgRPM") }}</span>
          <span class="stat-value">{{ avgRPM }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="rpmSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col amber">
          <Flame :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">{{ t("dashboard.avgTPM") }}</span>
          <span class="stat-value">{{ avgTPM }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="tpmSparkline" />
          </svg>
        </div>
      </div>

    </div>

    <div class="dashboard-grid">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <ReceiptText :size="20" />
            {{ t("dashboard.recentUsage") }}
          </h3>
          <el-button type="primary" link @click="router.push('/user/usage')"> {{ t("dashboard.viewAll") }} </el-button>
        </div>
        <div class="card-body">
          <div v-if="recentUsageLogs.length === 0" class="empty-state">
            <div class="empty-icon">
              <ReceiptText :size="32" />
            </div>
            <h4 class="empty-title">{{ t("dashboard.noUsageRecords") }}</h4>
            <p class="empty-description">{{ t("dashboard.noUsageRecordsDesc") }}</p>
          </div>
          <div v-else class="usage-list">
            <div v-for="item in recentUsageLogs.slice(0, 5)" :key="item.id" class="usage-item">
              <div class="usage-header">
                <span class="usage-model">{{ item.model }}</span>
                <el-tag size="small">{{ item.provider }}</el-tag>
              </div>
              <p class="usage-endpoint mono">{{ item.endpoint }}</p>
              <div class="usage-stats">
                <span>{{ t("common.input") }} {{ item.input_tokens }}</span>
                <span>{{ t("common.output") }} {{ item.output_tokens }}</span>
                <span class="usage-cost">{{ t("dashboard.costWithCurrency", { amount: formatCurrency(item.cost) }) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="dashboard-side">
        <div class="surface-card">
          <div class="card-header">
            <h3 class="card-title">
              <WalletCards :size="20" />
              {{ t("dashboard.latestOrders") }}
            </h3>
            <el-button type="primary" link @click="router.push('/user/payment')"> {{ t("dashboard.recharge") }} </el-button>
          </div>
          <div class="card-body">
            <div v-if="!latestOrder" class="empty-state">
              <div class="empty-icon">
                <WalletCards :size="32" />
              </div>
              <h4 class="empty-title">{{ t("dashboard.noOrders") }}</h4>
              <p class="empty-description">{{ t("dashboard.noOrdersDesc") }}</p>
            </div>
            <div v-else class="order-info">
              <div class="order-detail">
                <span class="order-label">{{ t("dashboard.merchantOrderNo") }}</span>
                <span class="order-value mono">{{ latestOrder.out_trade_no }}</span>
              </div>
              <div class="order-detail">
                <span class="order-label">{{ t("dashboard.amount") }}</span>
                <span class="order-value">{{ formatCurrency(latestOrder.amount) }} {{ t("common.currency") }}</span>
              </div>
              <div class="order-detail">
                <span class="order-label">{{ t("dashboard.orderStatus") }}</span>
                <el-tag :type="latestOrder.status === 'paid' ? 'success' : 'warning'" size="small">
                  {{ latestOrder.status === "paid" ? t("dashboard.paid") : t("dashboard.pending") }}
                </el-tag>
              </div>
              <div class="order-detail">
                <span class="order-label">{{ t("dashboard.createTime") }}</span>
                <span class="order-value">{{ formatTime(latestOrder.created_at_ms) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="surface-card">
          <div class="card-header">
            <h3 class="card-title">
              <Bolt :size="20" />
              {{ t("dashboard.quickActions") }}
            </h3>
          </div>
          <div class="card-body">
            <div class="quick-actions">
              <el-button type="primary" size="large" @click="router.push('/user/keys')">
                <KeyRound :size="18" />
                {{ t('dashboard.createApiKey') }}
              </el-button>
              <el-button type="warning" size="large" @click="router.push('/user/payment')">
                <WalletCards :size="18" />
                {{ t('dashboard.rechargeNow') }}
              </el-button>
              <el-button type="info" size="large" @click="router.push('/user/access-guide')">
                <BookOpenText :size="18" />
                {{ t('dashboard.accessGuide') }}
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { BarChart3, Bolt, BookOpenText, KeyRound, ReceiptText, Send, Timer, TrendingDown, Type, WalletCards, Coins, Flame } from "lucide-vue-next";
import { ElButton, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import type { PaymentOrder, UsageLog, UserDashboardResponse } from "@/api/types";
import { formatCurrency, formatNumber, formatTime } from "@/utils";

const { t } = useI18n();
const router = useRouter();
const dashboard = ref<UserDashboardResponse | null>(null);

const balance = computed(() => dashboard.value?.balance || 0);
const totalCost = computed(() => dashboard.value?.total_cost || 0);
const requestCount = computed(() => dashboard.value?.request_count || 0);
const recentUsageCount = computed(() => dashboard.value?.recent_usage_count || 0);
const totalTokens = computed(() => dashboard.value?.total_tokens || 0);
const avgRPM = computed(() => dashboard.value?.avg_rpm || "0");
const avgTPM = computed(() => dashboard.value?.avg_tpm || "0");
const recentUsageLogs = computed<UsageLog[]>(() => dashboard.value?.recent_usage_logs || []);
const recentPaymentOrders = computed<PaymentOrder[]>(() => dashboard.value?.recent_payment_orders || []);
const latestOrder = computed(() => recentPaymentOrders.value[0] || null);

// Sparkline 生成函数
function generateSparkline(data: number[]): string {
  if (!data.length) return "";
  const max = Math.max(...data, 1);
  const min = Math.min(...data);
  const range = max - min || 1;
  const step = 100 / (data.length - 1 || 1);
  return data
    .map((val, i) => {
      const x = i * step;
      const y = 30 - ((val - min) / range) * 28 - 1;
      return `${x},${y}`;
    })
    .join(" ");
}

const requestTimeline = computed(() => dashboard.value?.usage_timeline || []);
const costTimeline = computed(() => dashboard.value?.cost_timeline || []);
const tokenTimeline = computed(() => dashboard.value?.token_timeline || []);

const requestSparkline = computed(() => generateSparkline(requestTimeline.value));
const recentSparkline = computed(() => generateSparkline(requestTimeline.value.slice(-7)));
const costSparkline = computed(() => generateSparkline(costTimeline.value));
const tokenSparkline = computed(() => generateSparkline(tokenTimeline.value));
const rpmSparkline = computed(() => generateSparkline(requestTimeline.value.map((count) => count / 1440)));
const tpmSparkline = computed(() => generateSparkline(tokenTimeline.value.map((count) => count / 1440)));

async function load() {
  try {
    dashboard.value = await userAPI.dashboard();
  } catch {
    dashboard.value = null;
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--bg-raised);
  border-radius: var(--radius-xl);
  padding: 16px 18px;
  box-shadow: var(--shadow-sm);
  transition: all var(--transition-normal);
  border: 1px solid var(--border-default);
  position: relative;
  overflow: hidden;
  cursor: default;
  display: flex;
  align-items: center;
  gap: 12px;
}

.stat-card::after {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: linear-gradient(90deg, var(--primary-color), var(--accent-color));
  opacity: 0;
  transition: opacity var(--transition-normal);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
  border-color: var(--border-focus);
}

.stat-card:hover::after {
  opacity: 1;
}

.stat-icon-col {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: white;
}

.stat-icon-col.blue {
  background: linear-gradient(135deg, #3b82f6, #60a5fa);
}
.stat-icon-col.purple {
  background: linear-gradient(135deg, #8b5cf6, #a78bfa);
}
.stat-icon-col.green {
  background: linear-gradient(135deg, #10b981, #34d399);
}
.stat-icon-col.orange {
  background: linear-gradient(135deg, #f59e0b, #fbbf24);
}
.stat-icon-col.teal {
  background: linear-gradient(135deg, #14b8a6, #2dd4bf);
}
.stat-icon-col.cyan {
  background: linear-gradient(135deg, #06b6d4, #22d3ee);
}
.stat-icon-col.pink {
  background: linear-gradient(135deg, #ec4899, #f472b6);
}
.stat-icon-col.indigo {
  background: linear-gradient(135deg, #6366f1, #818cf8);
}
.stat-icon-col.yellow {
  background: linear-gradient(135deg, #eab308, #facc15);
}
.stat-icon-col.rose {
  background: linear-gradient(135deg, #f43f5e, #fb7185);
}
.stat-icon-col.emerald {
  background: linear-gradient(135deg, #059669, #10b981);
}
.stat-icon-col.sky {
  background: linear-gradient(135deg, #0ea5e9, #38bdf8);
}
.stat-icon-col.violet {
  background: linear-gradient(135deg, #7c3aed, #a78bfa);
}
.stat-icon-col.amber {
  background: linear-gradient(135deg, #d97706, #fbbf24);
}
.stat-icon-col.lime {
  background: linear-gradient(135deg, #65a30d, #a3e635);
}
.stat-icon-col.slate {
  background: linear-gradient(135deg, #475569, #94a3b8);
}

.stat-text-col {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}

.stat-label {
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 500;
  line-height: 1.3;
}

.stat-value {
  font-size: 20px;
  font-weight: 800;
  color: var(--text-primary);
  line-height: 1.2;
  margin: 0;
  letter-spacing: -0.03em;
}

.stat-action {
  margin-left: auto;
  flex-shrink: 0;
}

.stat-chart {
  width: 80px;
  height: 30px;
  flex-shrink: 0;
  color: var(--primary-color);
  opacity: 0.6;
}

.sparkline {
  width: 100%;
  height: 100%;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 24px;
  margin-top: 24px;
  margin-bottom: 24px;
}

.dashboard-side {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.usage-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.usage-item {
  padding: 16px;
  background: var(--border-light);
  border-radius: var(--radius-lg);
  transition: all var(--transition-fast);
}

.usage-item:hover {
  background: var(--border-subtle);
}

.usage-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.usage-model {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
}

.usage-endpoint {
  font-size: 12px;
  color: var(--text-muted);
  margin: 0 0 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.usage-stats {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: var(--text-secondary);
}

.usage-cost {
  color: var(--warning-color);
  font-weight: 500;
}

.order-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.order-detail {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.order-label {
  font-size: 13px;
  color: var(--text-muted);
}

.order-value {
  font-size: 14px;
  color: var(--text-primary);
  font-weight: 500;
}

.quick-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.quick-actions .el-button {
  justify-content: flex-start;
  margin: 0px !important;
}

.quick-actions .el-button :deep(svg) {
  margin-right: 10px;
}

@media (max-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
