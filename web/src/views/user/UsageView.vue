<template>
  <div class="usage-page">
    <div class="surface-card filter-bar">
      <div class="filter-left">
        <div class="filter-label">时间范围</div>
        <el-date-picker v-model="dateRange" type="daterange" range-separator="至" start-placeholder="开始日期" end-placeholder="结束日期" size="large" :shortcuts="dateShortcuts" value-format="x" />
        <el-input v-model="modelFilter" placeholder="模型 / codex" clearable size="large" class="model-filter" />
      </div>
      <div class="filter-right">
        <el-button-group>
          <el-button :type="rangeMode === 'today' ? 'primary' : ''" @click="setRange('today')">今天</el-button>
          <el-button :type="rangeMode === 'week' ? 'primary' : ''" @click="setRange('week')">7天</el-button>
          <el-button :type="rangeMode === 'month' ? 'primary' : ''" @click="setRange('month')">30天</el-button>
        </el-button-group>
      </div>
    </div>

    <div class="stat-row">
      <div class="surface-card stat-mini">
        <div class="stat-mini-icon requests">
          <Bolt :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">总调用</span>
          <span class="stat-mini-value">{{ filteredUsage.length }}</span>
        </div>
      </div>
      <div class="surface-card stat-mini">
        <div class="stat-mini-icon tokens-in">
          <ArrowDownToLine :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">输入 TOKEN</span>
          <span class="stat-mini-value">{{ formatNumber(summary.input) }}</span>
        </div>
      </div>
      <div class="surface-card stat-mini">
        <div class="stat-mini-icon tokens-out">
          <ArrowUpFromLine :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">输出 TOKEN</span>
          <span class="stat-mini-value">{{ formatNumber(summary.output) }}</span>
        </div>
      </div>
      <div class="surface-card stat-mini">
        <div class="stat-mini-icon cost">
          <ReceiptText :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">累计花费</span>
          <span class="stat-mini-value cost-value">{{ formatCurrency(summary.cost) }}</span>
        </div>
      </div>
    </div>

    <div class="usage-grid">
      <div class="surface-card quota-section">
        <div class="card-header">
          <h3 class="card-title">
            <Gauge :size="20" />
            用量限额
          </h3>
        </div>
        <div class="card-body">
          <div v-if="!modelStats.length" class="empty-hint">暂无数据</div>
          <div v-else class="quota-list">
            <div v-for="item in topModels" :key="item.model" class="quota-item">
              <div class="quota-info">
                <span class="quota-model">{{ item.model }}</span>
                <span class="quota-provider">{{ item.provider }}</span>
              </div>
              <div class="quota-bars">
                <div class="quota-bar-wrap">
                  <span class="bar-label">输入</span>
                  <div class="bar-track">
                    <div class="bar-fill bar-input" :style="{ width: barWidth(item.inputPct) }"></div>
                  </div>
                  <span class="bar-value">{{ item.input.toLocaleString() }} / {{ item.inputMax }}</span>
                </div>
                <div class="quota-bar-wrap">
                  <span class="bar-label">输出</span>
                  <div class="bar-track">
                    <div class="bar-fill bar-output" :style="{ width: barWidth(item.outputPct) }"></div>
                  </div>
                  <span class="bar-value">{{ item.output.toLocaleString() }} / {{ item.outputMax }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="surface-card dist-section">
        <div class="card-header">
          <h3 class="card-title">
            <PieChart :size="20" />
            Token 使用分布
          </h3>
        </div>
        <div class="card-body">
          <div v-if="!providerStats.length" class="empty-hint">暂无数据</div>
          <div v-else class="dist-list">
            <div v-for="item in providerStats" :key="item.name" class="dist-item">
              <div class="dist-header">
                <span class="dist-name">{{ item.name }}</span>
                <span class="dist-pct">{{ item.pct }}%</span>
              </div>
              <div class="dist-bar-track">
                <div class="dist-bar-fill" :style="{ width: item.pct + '%' }" :class="`dist-color-${item.idx % 5}`"></div>
              </div>
              <div class="dist-detail">
                <span>调用 {{ item.count }} 次</span>
                <span>Token {{ formatNumber(item.tokens) }}</span>
                <span class="dist-cost">{{ formatCurrency(item.cost) }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="usage-grid">
      <div class="surface-card detail-section">
        <div class="card-header">
          <h3 class="card-title">
            <ListOrdered :size="20" />
            详细统计数据
          </h3>
        </div>
        <div class="card-body">
          <div v-if="!filteredUsage.length" class="empty-hint">暂无数据</div>
          <el-table v-else :data="filteredUsage.slice(0, 10)" empty-text="暂无记录" class="modern-table" :stripe="true">
            <el-table-column prop="model" label="模型" min-width="160" />
            <el-table-column prop="provider" label="渠道" width="100">
              <template #default="{ row }">
                <el-tag size="small">{{ row.provider }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="输入 Token" width="110" align="right">
              <template #default="{ row }">
                <span class="token-num">{{ row.input_tokens.toLocaleString() }}</span>
              </template>
            </el-table-column>
            <el-table-column label="输出 Token" width="110" align="right">
              <template #default="{ row }">
                <span class="token-num">{{ row.output_tokens.toLocaleString() }}</span>
              </template>
            </el-table-column>
            <el-table-column label="费用" width="100" align="right">
              <template #default="{ row }">
                <span class="cost-text">{{ formatCurrency(row.cost) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="时间" min-width="170">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.created_at_ms) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <div class="surface-card trend-section">
        <div class="card-header">
          <h3 class="card-title">
            <TrendingUp :size="20" />
            调用趋势
          </h3>
        </div>
        <div class="card-body">
          <div v-if="!dailyData.length" class="empty-hint">暂无数据</div>
          <div v-else class="trend-chart">
            <div class="chart-bars">
              <div v-for="(day, i) in dailyData" :key="day.date" class="chart-col" :title="`${day.date}: ${day.count} 次调用`">
                <div class="bar-container">
                  <div class="bar-fill-trend" :style="{ height: trendBarHeight(day.count) }"></div>
                </div>
                <span class="bar-day">{{ day.label }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ArrowDownToLine, ArrowUpFromLine, Bolt, Gauge, ListOrdered, PieChart, ReceiptText, TrendingUp } from "lucide-vue-next";
import { ElButton, ElButtonGroup, ElDatePicker, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import type { UsageLog } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const usage = ref<UsageLog[]>([]);
const dateRange = ref<number[] | null>(null);
const modelFilter = ref("");
const rangeMode = ref("today");

const dateShortcuts = [
  {
    text: "最近一周",
    value: () => {
      const e = new Date();
      const s = new Date();
      s.setDate(s.getDate() - 7);
      return [s, e];
    },
  },
  {
    text: "最近一月",
    value: () => {
      const e = new Date();
      const s = new Date();
      s.setMonth(s.getMonth() - 1);
      return [s, e];
    },
  },
  {
    text: "最近三月",
    value: () => {
      const e = new Date();
      const s = new Date();
      s.setMonth(s.getMonth() - 3);
      return [s, e];
    },
  },
];

function setRange(mode: string) {
  rangeMode.value = mode;
  const now = Date.now();
  const day = 86400000;
  if (mode === "today") {
    const start = new Date();
    start.setHours(0, 0, 0, 0);
    dateRange.value = [start.getTime(), now];
  } else if (mode === "week") {
    dateRange.value = [now - 7 * day, now];
  } else if (mode === "month") {
    dateRange.value = [now - 30 * day, now];
  }
}

const filteredUsage = computed(() => {
  let list = usage.value;
  if (dateRange.value && dateRange.value[0] && dateRange.value[1]) {
    list = list.filter((item) => item.created_at_ms >= dateRange.value![0] && item.created_at_ms <= dateRange.value![1]);
  }
  if (modelFilter.value.trim()) {
    const q = modelFilter.value.toLowerCase();
    list = list.filter((item) => item.model.toLowerCase().includes(q));
  }
  return list;
});

const summary = computed(() => ({
  requests: filteredUsage.value.length,
  input: filteredUsage.value.reduce((sum, item) => sum + item.input_tokens, 0),
  output: filteredUsage.value.reduce((sum, item) => sum + item.output_tokens, 0),
  cost: filteredUsage.value.reduce((sum, item) => sum + item.cost, 0),
}));

interface ModelStat {
  model: string;
  provider: string;
  input: number;
  output: number;
  count: number;
  inputMax: number;
  outputMax: number;
  inputPct: number;
  outputPct: number;
}

const modelStats = computed((): ModelStat[] => {
  const map = new Map<string, ModelStat>();
  for (const item of filteredUsage.value) {
    const key = item.model;
    if (!map.has(key)) {
      map.set(key, { model: item.model, provider: item.provider, input: 0, output: 0, count: 0, inputMax: 200000, outputMax: 80000, inputPct: 0, outputPct: 0 });
    }
    const s = map.get(key)!;
    s.input += item.input_tokens;
    s.output += item.output_tokens;
    s.count++;
  }
  const result = Array.from(map.values());
  result.forEach((s) => {
    s.inputPct = Math.min(100, (s.input / s.inputMax) * 100);
    s.outputPct = Math.min(100, (s.output / s.outputMax) * 100);
  });
  return result.sort((a, b) => b.count - a.count);
});

const topModels = computed(() => modelStats.value.slice(0, 6));

interface ProviderStat {
  name: string;
  count: number;
  tokens: number;
  cost: number;
  pct: number;
  idx: number;
}

const totalTokens = computed(() => summary.value.input + summary.value.output);

const providerStats = computed((): ProviderStat[] => {
  const map = new Map<string, { count: number; tokens: number; cost: number }>();
  for (const item of filteredUsage.value) {
    if (!map.has(item.provider)) map.set(item.provider, { count: 0, tokens: 0, cost: 0 });
    const s = map.get(item.provider)!;
    s.count++;
    s.tokens += item.input_tokens + item.output_tokens;
    s.cost += item.cost;
  }
  const result = Array.from(map.entries()).map(([name, v], idx) => ({
    name,
    ...v,
    pct: totalTokens.value ? Math.round((v.tokens / totalTokens.value) * 1000) / 10 : 0,
    idx,
  }));
  return result.sort((a, b) => b.count - a.count);
});

interface DayData {
  date: string;
  label: string;
  count: number;
}

const dailyData = computed((): DayData[] => {
  const map = new Map<string, number>();
  for (const item of filteredUsage.value) {
    const d = new Date(item.created_at_ms);
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
    map.set(key, (map.get(key) || 0) + 1);
  }
  const entries = Array.from(map.entries()).sort();
  return entries.map(([date, count]) => ({
    date,
    label: date.slice(5),
    count,
  }));
});

const maxDailyCount = computed(() => Math.max(...dailyData.value.map((d) => d.count), 1));

function formatNumber(n: number): string {
  if (n >= 1e6) return (n / 1e6).toFixed(1) + "M";
  if (n >= 1e3) return (n / 1e3).toFixed(1) + "K";
  return String(n);
}

function barWidth(pct: number): string {
  return Math.max(pct, 2) + "%";
}

function trendBarHeight(count: number): string {
  const maxH = 120;
  return Math.max((count / maxDailyCount.value) * maxH, 4) + "px";
}

async function load() {
  usage.value = await userAPI.usage();
}

onMounted(() => {
  setRange("today");
  void load();
});
</script>

<style scoped>
.usage-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
  padding: 0px 16px;
}

.filter-left {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.filter-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
  white-space: nowrap;
}

.model-filter {
  width: 180px !important;
}

.filter-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  width: 100%;
}

.stat-row .surface-card {
  min-width: 0;
  width: 100%;
}

.stat-mini {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px !important;
}

.stat-mini-icon {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-mini-icon.requests {
  background: var(--primary-lighter);
  color: var(--primary-color);
}
.stat-mini-icon.tokens-in {
  background: var(--info-light);
  color: var(--info-color);
}
.stat-mini-icon.tokens-out {
  background: var(--success-light);
  color: var(--success-color);
}
.stat-mini-icon.cost {
  background: var(--warning-light);
  color: var(--warning-color);
}

.stat-mini-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-mini-label {
  font-size: 11.5px;
  color: var(--text-muted);
  font-weight: 500;
}

.stat-mini-value {
  font-size: 22px;
  font-weight: 800;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

.cost-value {
  color: var(--warning-color);
}

.usage-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.quota-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.quota-item {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--border-subtle);
}

.quota-item:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

.quota-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.quota-model {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.quota-provider {
  font-size: 12px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 2px 8px;
  border-radius: var(--radius-sm);
}

.quota-bars {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.quota-bar-wrap {
  display: grid;
  grid-template-columns: 36px 1fr auto;
  align-items: center;
  gap: 10px;
}

.bar-label {
  font-size: 11.5px;
  color: var(--text-muted);
  font-weight: 600;
}

.bar-track {
  height: 8px;
  background: var(--border-light);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.5s ease;
}

.bar-input {
  background: linear-gradient(90deg, var(--info-color), var(--primary-color));
}
.bar-output {
  background: linear-gradient(90deg, var(--success-color), var(--accent-color));
}

.bar-value {
  font-size: 11.5px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  white-space: nowrap;
}

.dist-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.dist-item {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--border-subtle);
}

.dist-item:last-child {
  padding-bottom: 0;
  border-bottom: none;
}

.dist-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.dist-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.dist-pct {
  font-size: 15px;
  font-weight: 800;
  color: var(--primary-color);
}

.dist-bar-track {
  height: 10px;
  background: var(--border-light);
  border-radius: var(--radius-full);
  overflow: hidden;
  margin-bottom: 8px;
}

.dist-bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.5s ease;
}

.dist-color-0 {
  background: linear-gradient(90deg, hsl(234, 70%, 65%), hsl(262, 60%, 60%));
}
.dist-color-1 {
  background: linear-gradient(90deg, hsl(160, 60%, 50%), hsl(150, 55%, 45%));
}
.dist-color-2 {
  background: linear-gradient(90deg, hsl(38, 90%, 55%), hsl(35, 85%, 50%));
}
.dist-color-3 {
  background: linear-gradient(90deg, hsl(340, 75%, 55%), hsl(330, 70%, 50%));
}
.dist-color-4 {
  background: linear-gradient(90deg, hsl(280, 55%, 60%), hsl(270, 50%, 55%));
}

.dist-detail {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--text-muted);
}

.dist-cost {
  margin-left: auto;
  color: var(--warning-color);
  font-weight: 600;
}

.trend-chart {
  padding: 10px 0;
}

.chart-bars {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  height: 150px;
  padding-top: 10px;
}

.chart-col {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  min-width: 24px;
}

.bar-container {
  width: 100%;
  height: 120px;
  display: flex;
  align-items: flex-end;
  justify-content: center;
}

.bar-fill-trend {
  width: 80%;
  max-width: 32px;
  min-height: 4px;
  border-radius: 4px 4px 0 0;
  background: linear-gradient(180deg, var(--primary-color), var(--primary-light));
  transition: height 0.4s ease;
}

.bar-day {
  font-size: 10px;
  color: var(--text-muted);
  white-space: nowrap;
}

.empty-hint {
  text-align: center;
  padding: 48px 20px;
  color: var(--text-muted);
  font-size: 14px;
}

.token-num {
  font-family: var(--font-mono);
  font-size: 13px;
}

.cost-text {
  color: var(--warning-color);
  font-weight: 600;
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}

@media (max-width: 900px) {
  .usage-grid {
    grid-template-columns: 1fr;
  }

  .filter-left,
  .filter-right {
    width: 100%;
  }

  .model-filter {
    width: 100% !important;
  }
}
</style>
