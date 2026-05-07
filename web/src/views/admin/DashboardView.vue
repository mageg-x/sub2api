<template>
  <div>
    <div v-if="error" class="error-banner">
      <el-alert :title="error" type="error" :closable="false" show-icon />
    </div>

    <div class="card-grid">
      <div v-for="item in metricCards" :key="item.label" class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <component :is="item.icon" :size="24" />
          </div>
          <span v-if="item.trend" :class="['stat-trend', item.trend > 0 ? 'up' : 'down']"> {{ item.trend > 0 ? "+" : "" }}{{ item.trend }}% </span>
        </div>
        <p class="stat-label">{{ item.label }}</p>
        <p class="stat-value">{{ item.value }}</p>
        <p class="stat-helper">{{ item.helper }}</p>
      </div>
    </div>

    <div class="dashboard-grid">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <CircleDollarSign :size="20" />
            系统指标
          </h3>
        </div>
        <div class="card-body">
          <div v-if="!loading && statEntries.length === 0" class="empty-state">
            <div class="empty-icon">
              <Gauge :size="32" />
            </div>
            <h4 class="empty-title">暂无指标</h4>
            <p class="empty-description">系统指标将在这里显示</p>
          </div>
          <div v-else class="stats-grid">
            <div v-for="entry in statEntries" :key="entry[0]" class="stat-item">
              <div class="stat-item-label">{{ entry[0] }}</div>
              <div :class="['stat-item-value', { 'date-value': entry[2] }]">{{ entry[1] }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <Bell :size="20" />
            最新公告
          </h3>
          <el-button type="primary" link @click="router.push('/admin/announcements')"> 查看全部 </el-button>
        </div>
        <div class="card-body">
          <div v-if="!data?.announcements?.length" class="empty-state">
            <div class="empty-icon">
              <Bell :size="32" />
            </div>
            <h4 class="empty-title">暂无公告</h4>
            <p class="empty-description">发布一条公告吧</p>
          </div>
          <div v-else class="announcement-list">
            <div v-for="item in data.announcements.slice(0, 4)" :key="item.id" class="announcement-item">
              <div class="announcement-header">
                <h4 class="announcement-title">{{ item.title }}</h4>
                <el-tag :type="item.status === 'active' ? 'success' : 'info'" size="small">
                  {{ item.status }}
                </el-tag>
              </div>
              <p class="announcement-content">{{ item.content }}</p>
              <p class="announcement-time">{{ formatTime(item.published_at_ms) }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Boxes :size="20" />
          上游账户池
        </h3>
        <el-button type="primary" link @click="router.push('/admin/accounts')"> 管理账户 </el-button>
      </div>
      <div class="card-body">
        <el-table :data="data?.accounts?.slice(0, 8) || []" empty-text="暂无账户" class="data-table">
          <el-table-column prop="provider" label="Provider" width="120" />
          <el-table-column prop="name" label="账户名" min-width="140" />
          <el-table-column prop="auth_type" label="认证方式" width="100" />
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="isActiveStatus(row.status) ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <ListOrdered :size="20" />
          最新订单
        </h3>
        <el-button type="primary" link @click="router.push('/admin/payments')"> 查看全部 </el-button>
      </div>
      <div class="card-body">
        <el-table :data="data?.orders?.slice(0, 8) || []" empty-text="暂无订单" class="data-table">
          <el-table-column prop="out_trade_no" label="商户单号" min-width="160">
            <template #default="{ row }">
              <span class="mono">{{ row.out_trade_no }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="user_id" label="用户" width="60" />
          <el-table-column prop="provider" label="渠道" width="80" />
          <el-table-column label="金额" width="80">
            <template #default="{ row }">
              <span class="mono">{{ formatCurrency(row.amount) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="70">
            <template #default="{ row }">
              <el-tag :type="isPaidStatus(row.status) ? 'success' : 'warning'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="130">
            <template #default="{ row }">
              <span class="mono">{{ formatTime(row.created_at_ms) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Bell, Boxes, CircleDollarSign, Gauge, ListOrdered, Users } from "lucide-vue-next";
import { ElAlert, ElButton, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { DashboardResponse } from "@/api/types";
import { formatCurrency, formatTime, isActiveStatus, isPaidStatus } from "@/utils";

const router = useRouter();

const data = ref<DashboardResponse | null>(null);
const loading = ref(false);
const error = ref("");

interface MetricCard {
  label: string;
  value: string | number;
  helper: string;
  icon: unknown;
  trend: number | null;
}

const metricCards = computed<MetricCard[]>(() => {
  if (!data.value) {
    return [
      { label: "用户数", value: "-", helper: "平台注册用户", icon: Users, trend: null },
      { label: "上游账户", value: "-", helper: "OAuth / 静态密钥", icon: Boxes, trend: null },
      { label: "支付订单", value: "-", helper: "一期仅 gopay", icon: ListOrdered, trend: null },
    ];
  }

  return [
    {
      label: "用户数",
      value: data.value.users.length,
      helper: "平台注册用户",
      icon: Users,
      trend: null,
    },
    {
      label: "上游账户",
      value: data.value.accounts.length,
      helper: "OAuth / 静态密钥",
      icon: Boxes,
      trend: null,
    },
    {
      label: "支付订单",
      value: data.value.orders.length,
      helper: "一期仅 gopay",
      icon: ListOrdered,
      trend: null,
    },
  ];
});

const statEntries = computed<Array<[string, unknown, boolean]>>(() => {
  const raw = Object.entries(data.value?.stats || {});
  return raw
    .filter(([key]) => key !== "TIMESTAMP_MS")
    .map(([key, value]) => {
      if (typeof value === "number" && value > 1e12) {
        return [key, formatTime(value), true];
      }
      return [key, value, false];
    });
});

async function load() {
  loading.value = true;
  error.value = "";
  try {
    data.value = await adminAPI.dashboard();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "加载失败";
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.error-banner {
  margin-bottom: 24px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 24px;
  margin-bottom: 24px;
}

.stats-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 12px;
}

.stat-item {
  padding: 14px;
  background: var(--border-light);
  border-radius: var(--radius-md);
  transition: all var(--transition-fast);
  overflow: hidden;
}

.stat-item:hover {
  background: var(--border-subtle);
}

.stat-item-label {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.stat-item-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.02em;
  word-break: break-all;
  line-height: 1.3;
}

.stat-item-value.date-value {
  font-size: 13px;
  font-weight: 700;
  color: var(--text-primary);
  line-height: 1.4;
}

.announcement-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.announcement-item {
  padding: 16px;
  background: var(--border-light);
  border-radius: var(--radius-lg);
  transition: all var(--transition-fast);
}

.announcement-item:hover {
  background: var(--border-subtle);
}

.announcement-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.announcement-title {
  font-size: 15px;
  font-weight: 600;
  margin: 0;
  color: var(--text-primary);
}

.announcement-content {
  font-size: 13px;
  color: var(--text-secondary);
  margin: 0 0 8px;
  line-height: 1.5;
}

.announcement-time {
  font-size: 12px;
  color: var(--text-muted);
  margin: 0;
}

@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>
