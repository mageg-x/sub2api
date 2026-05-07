<template>
  <div>
    <!-- 账户数据 -->
    <div class="section-title">
      <WalletCards :size="18" />
      <span>账户数据</span>
    </div>
    <div class="card-grid">
      <div class="stat-card">
        <div class="stat-icon-col blue">
          <WalletCards :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">当前余额</span>
          <span class="stat-value">{{ formatCurrency(session.user?.balance || 0) }}</span>
        </div>
        <el-button type="primary" size="small" class="stat-action" @click="router.push('/user/payment')">
          充值
        </el-button>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col purple">
          <TrendingDown :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">历史消耗</span>
          <span class="stat-value">{{ formatCurrency(totalCost) }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col green">
          <CircleDollarSign :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">累计充值</span>
          <span class="stat-value">{{ formatCurrency(totalRecharge) }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col orange">
          <Percent :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">费率折扣</span>
          <span class="stat-value">{{ session.user?.rate_percent || 100 }}%</span>
        </div>
      </div>
    </div>

    <!-- 使用统计 -->
    <div class="section-title">
      <Activity :size="18" />
      <span>使用统计</span>
    </div>
    <div class="card-grid">
      <div class="stat-card">
        <div class="stat-icon-col teal">
          <Send :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">请求次数</span>
          <span class="stat-value">{{ usage.length }}</span>
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
          <span class="stat-label">统计次数(7天)</span>
          <span class="stat-value">{{ recentUsageCount }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="recentSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col pink">
          <KeyRound :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">API Keys</span>
          <span class="stat-value">{{ keyCount }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col indigo">
          <Clock :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">最近调用</span>
          <span class="stat-value">{{ lastUsageTime }}</span>
        </div>
      </div>
    </div>

    <!-- 资源消耗 -->
    <div class="section-title">
      <Zap :size="18" />
      <span>资源消耗</span>
    </div>
    <div class="card-grid">
      <div class="stat-card">
        <div class="stat-icon-col yellow">
          <Coins :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">统计额度</span>
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
          <span class="stat-label">统计 Tokens</span>
          <span class="stat-value">{{ formatNumber(totalTokens) }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="tokenSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col emerald">
          <ArrowDownToLine :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">输入 Tokens</span>
          <span class="stat-value">{{ formatNumber(totalInputTokens) }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col sky">
          <ArrowUpFromLine :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">输出 Tokens</span>
          <span class="stat-value">{{ formatNumber(totalOutputTokens) }}</span>
        </div>
      </div>
    </div>

    <!-- 性能指标 -->
    <div class="section-title">
      <Gauge :size="18" />
      <span>性能指标</span>
    </div>
    <div class="card-grid">
      <div class="stat-card">
        <div class="stat-icon-col violet">
          <Timer :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">平均 RPM</span>
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
          <span class="stat-label">平均 TPM</span>
          <span class="stat-value">{{ avgTPM }}</span>
        </div>
        <div class="stat-chart">
          <svg viewBox="0 0 100 30" class="sparkline">
            <polyline fill="none" stroke="currentColor" stroke-width="2" :points="tpmSparkline" />
          </svg>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col lime">
          <Layers :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">常用模型</span>
          <span class="stat-value">{{ topModel }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-col slate">
          <Globe :size="28" />
        </div>
        <div class="stat-text-col">
          <span class="stat-label">常用渠道</span>
          <span class="stat-value">{{ topProvider }}</span>
        </div>
      </div>
    </div>

    <div class="dashboard-grid">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <ReceiptText :size="20" />
            最近调用
          </h3>
          <el-button type="primary" link @click="router.push('/user/usage')"> 查看全部 </el-button>
        </div>
        <div class="card-body">
          <div v-if="usage.length === 0" class="empty-state">
            <div class="empty-icon">
              <ReceiptText :size="32" />
            </div>
            <h4 class="empty-title">暂无调用记录</h4>
            <p class="empty-description">您的 API 调用记录将显示在这里</p>
          </div>
          <div v-else class="usage-list">
            <div v-for="item in usage.slice(0, 5)" :key="item.id" class="usage-item">
              <div class="usage-header">
                <span class="usage-model">{{ item.model }}</span>
                <el-tag size="small">{{ item.provider }}</el-tag>
              </div>
              <p class="usage-endpoint mono">{{ item.endpoint }}</p>
              <div class="usage-stats">
                <span>输入 {{ item.input_tokens }}</span>
                <span>输出 {{ item.output_tokens }}</span>
                <span class="usage-cost">花费 {{ formatCurrency(item.cost) }} 元</span>
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
              最新订单
            </h3>
            <el-button type="primary" link @click="router.push('/user/payment')"> 充值 </el-button>
          </div>
          <div class="card-body">
            <div v-if="!latestOrder" class="empty-state">
              <div class="empty-icon">
                <WalletCards :size="32" />
              </div>
              <h4 class="empty-title">暂无订单</h4>
              <p class="empty-description">您的充值订单将显示在这里</p>
            </div>
            <div v-else class="order-info">
              <div class="order-detail">
                <span class="order-label">商户单号</span>
                <span class="order-value mono">{{ latestOrder.out_trade_no }}</span>
              </div>
              <div class="order-detail">
                <span class="order-label">金额</span>
                <span class="order-value">{{ formatCurrency(latestOrder.amount) }} 元</span>
              </div>
              <div class="order-detail">
                <span class="order-label">状态</span>
                <el-tag :type="latestOrder.status === 'paid' ? 'success' : 'warning'" size="small">
                  {{ latestOrder.status }}
                </el-tag>
              </div>
              <div class="order-detail">
                <span class="order-label">创建时间</span>
                <span class="order-value">{{ formatTime(latestOrder.created_at_ms) }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="surface-card">
          <div class="card-header">
            <h3 class="card-title">
              <Bolt :size="20" />
              快速操作
            </h3>
          </div>
          <div class="card-body">
            <div class="quick-actions">
              <el-button type="primary" size="large" @click="router.push('/user/keys')">
                <KeyRound :size="18" />
                创建 API Key
              </el-button>
              <el-button type="warning" size="large" @click="router.push('/user/payment')">
                <WalletCards :size="18" />
                立即充值
              </el-button>
              <el-button type="info" size="large" @click="router.push('/user/access-guide')">
                <BookOpenText :size="18" />
                接入指南
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
import {
  Activity,
  ArrowDownToLine,
  ArrowUpFromLine,
  BarChart3,
  Bolt,
  BookOpenText,
  CircleDollarSign,
  Clock,
  Coins,
  Flame,
  Gauge,
  Globe,
  KeyRound,
  Layers,
  Percent,
  ReceiptText,
  Send,
  Timer,
  TrendingDown,
  Type,
  WalletCards,
  Zap,
} from "lucide-vue-next";
import { ElButton, ElTag } from "element-plus";
import { session } from "@/store/session";
import { userAPI } from "@/api/user";
import type { PaymentOrder, UsageLog } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const router = useRouter();

const usage = ref<UsageLog[]>([]);
const orders = ref<PaymentOrder[]>([]);
const keyCount = ref(0);

const latestOrder = computed(() => orders.value[0] || null);

// 总消耗
const totalCost = computed(() => usage.value.reduce((sum, item) => sum + (item.cost || 0), 0));

// 累计充值
const totalRecharge = computed(() =>
  orders.value.filter((o) => o.status === "paid").reduce((sum, item) => sum + (item.amount || 0), 0)
);

// Token 统计
const totalInputTokens = computed(() => usage.value.reduce((sum, item) => sum + (item.input_tokens || 0), 0));
const totalOutputTokens = computed(() => usage.value.reduce((sum, item) => sum + (item.output_tokens || 0), 0));
const totalTokens = computed(() => totalInputTokens.value + totalOutputTokens.value);

// 最近7天调用次数
const sevenDaysAgo = computed(() => Date.now() - 7 * 24 * 60 * 60 * 1000);
const recentUsageCount = computed(() => usage.value.filter((item) => item.created_at_ms > sevenDaysAgo.value).length);

// 最近调用时间
const lastUsageTime = computed(() => {
  if (!usage.value.length) return "-";
  return formatTime(usage.value[0].created_at_ms);
});

// 常用模型
const topModel = computed(() => {
  if (!usage.value.length) return "-";
  const counts: Record<string, number> = {};
  usage.value.forEach((item) => {
    counts[item.model] = (counts[item.model] || 0) + 1;
  });
  return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0] || "-";
});

// 常用渠道
const topProvider = computed(() => {
  if (!usage.value.length) return "-";
  const counts: Record<string, number> = {};
  usage.value.forEach((item) => {
    counts[item.provider] = (counts[item.provider] || 0) + 1;
  });
  return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0] || "-";
});

// RPM (Requests Per Minute) - 基于最近1小时
const avgRPM = computed(() => {
  if (!usage.value.length) return "0";
  const oneHourAgo = Date.now() - 60 * 60 * 1000;
  const recent = usage.value.filter((item) => item.created_at_ms > oneHourAgo);
  if (!recent.length) return "0";
  return (recent.length / 60).toFixed(3);
});

// TPM (Tokens Per Minute) - 基于最近1小时
const avgTPM = computed(() => {
  if (!usage.value.length) return "0";
  const oneHourAgo = Date.now() - 60 * 60 * 1000;
  const recent = usage.value.filter((item) => item.created_at_ms > oneHourAgo);
  if (!recent.length) return "0";
  const tokens = recent.reduce((sum, item) => sum + (item.input_tokens || 0) + (item.output_tokens || 0), 0);
  return (tokens / 60).toFixed(3);
});

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

// 按天分组的请求数 sparkline（最近14天）
const requestSparkline = computed(() => {
  const days = 14;
  const daily: number[] = new Array(days).fill(0);
  const now = Date.now();
  usage.value.forEach((item) => {
    const dayIndex = Math.floor((now - item.created_at_ms) / (24 * 60 * 60 * 1000));
    if (dayIndex >= 0 && dayIndex < days) {
      daily[days - 1 - dayIndex]++;
    }
  });
  return generateSparkline(daily);
});

// 最近7天 sparkline
const recentSparkline = computed(() => {
  const days = 7;
  const daily: number[] = new Array(days).fill(0);
  const now = Date.now();
  usage.value.forEach((item) => {
    const dayIndex = Math.floor((now - item.created_at_ms) / (24 * 60 * 60 * 1000));
    if (dayIndex >= 0 && dayIndex < days) {
      daily[days - 1 - dayIndex]++;
    }
  });
  return generateSparkline(daily);
});

// 成本 sparkline
const costSparkline = computed(() => {
  const days = 14;
  const daily: number[] = new Array(days).fill(0);
  const now = Date.now();
  usage.value.forEach((item) => {
    const dayIndex = Math.floor((now - item.created_at_ms) / (24 * 60 * 60 * 1000));
    if (dayIndex >= 0 && dayIndex < days) {
      daily[days - 1 - dayIndex] += item.cost || 0;
    }
  });
  return generateSparkline(daily);
});

// Token sparkline
const tokenSparkline = computed(() => {
  const days = 14;
  const daily: number[] = new Array(days).fill(0);
  const now = Date.now();
  usage.value.forEach((item) => {
    const dayIndex = Math.floor((now - item.created_at_ms) / (24 * 60 * 60 * 1000));
    if (dayIndex >= 0 && dayIndex < days) {
      daily[days - 1 - dayIndex] += (item.input_tokens || 0) + (item.output_tokens || 0);
    }
  });
  return generateSparkline(daily);
});

// RPM sparkline (最近12小时，每小时)
const rpmSparkline = computed(() => {
  const hours = 12;
  const hourly: number[] = new Array(hours).fill(0);
  const now = Date.now();
  usage.value.forEach((item) => {
    const hourIndex = Math.floor((now - item.created_at_ms) / (60 * 60 * 1000));
    if (hourIndex >= 0 && hourIndex < hours) {
      hourly[hours - 1 - hourIndex]++;
    }
  });
  return generateSparkline(hourly.map((c) => c / 60));
});

// TPM sparkline
const tpmSparkline = computed(() => {
  const hours = 12;
  const hourly: number[] = new Array(hours).fill(0);
  const now = Date.now();
  usage.value.forEach((item) => {
    const hourIndex = Math.floor((now - item.created_at_ms) / (60 * 60 * 1000));
    if (hourIndex >= 0 && hourIndex < hours) {
      hourly[hours - 1 - hourIndex] += (item.input_tokens || 0) + (item.output_tokens || 0);
    }
  });
  return generateSparkline(hourly.map((c) => c / 60));
});

function formatNumber(value: number): string {
  if (value >= 1e8) return (value / 1e8).toFixed(2) + "亿";
  if (value >= 1e4) return (value / 1e4).toFixed(2) + "万";
  return value.toLocaleString();
}

async function load() {
  usage.value = await userAPI.usage(100);
  orders.value = await userAPI.orders();
  try {
    const keys = await userAPI.keys();
    keyCount.value = keys.length;
  } catch {
    keyCount.value = 0;
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 24px 0 12px;
  padding-left: 4px;
}

.section-title:first-child {
  margin-top: 0;
}

.card-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 0;
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
  content: '';
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

.stat-icon-col.blue { background: linear-gradient(135deg, #3b82f6, #60a5fa); }
.stat-icon-col.purple { background: linear-gradient(135deg, #8b5cf6, #a78bfa); }
.stat-icon-col.green { background: linear-gradient(135deg, #10b981, #34d399); }
.stat-icon-col.orange { background: linear-gradient(135deg, #f59e0b, #fbbf24); }
.stat-icon-col.teal { background: linear-gradient(135deg, #14b8a6, #2dd4bf); }
.stat-icon-col.cyan { background: linear-gradient(135deg, #06b6d4, #22d3ee); }
.stat-icon-col.pink { background: linear-gradient(135deg, #ec4899, #f472b6); }
.stat-icon-col.indigo { background: linear-gradient(135deg, #6366f1, #818cf8); }
.stat-icon-col.yellow { background: linear-gradient(135deg, #eab308, #facc15); }
.stat-icon-col.rose { background: linear-gradient(135deg, #f43f5e, #fb7185); }
.stat-icon-col.emerald { background: linear-gradient(135deg, #059669, #10b981); }
.stat-icon-col.sky { background: linear-gradient(135deg, #0ea5e9, #38bdf8); }
.stat-icon-col.violet { background: linear-gradient(135deg, #7c3aed, #a78bfa); }
.stat-icon-col.amber { background: linear-gradient(135deg, #d97706, #fbbf24); }
.stat-icon-col.lime { background: linear-gradient(135deg, #65a30d, #a3e635); }
.stat-icon-col.slate { background: linear-gradient(135deg, #475569, #94a3b8); }

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
  .card-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .card-grid {
    grid-template-columns: 1fr;
  }
}
</style>
