<template>
  <div class="page-container">
    <div class="content-grid">
      <div class="stats-row">
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon total">
            <ListOrdered :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">订单总数</span>
            <span class="stat-mini-value">{{ orders.length }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon paid">
            <CheckCircle :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">已支付</span>
            <span class="stat-mini-value">{{ paidOrders }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon pending">
            <Clock :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">待处理</span>
            <span class="stat-mini-value">{{ pendingOrders }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon revenue">
            <CircleDollarSign :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">总收入</span>
            <span class="stat-mini-value">{{ formatCurrency(totalRevenue) }} 元</span>
          </div>
        </div>
      </div>

      <div class="surface-card orders-section">
        <div class="card-header">
          <h3 class="card-title">
            <ReceiptText :size="20" />
            订单列表
          </h3>
          <span class="order-count">{{ orders.length }} 个订单</span>
        </div>
        <div class="card-body">
          <el-table :data="orders" empty-text="暂无订单" class="modern-table" :stripe="true">
            <el-table-column prop="out_trade_no" label="商户单号" min-width="180">
              <template #default="{ row }">
                <code class="trade-no mono">{{ row.out_trade_no }}</code>
              </template>
            </el-table-column>
            <el-table-column prop="user_id" label="用户" width="70">
              <template #default="{ row }">
                <span class="user-id">#{{ row.user_id }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="provider" label="渠道" width="90">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.provider || "gopay" }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="金额" width="90">
              <template #default="{ row }">
                <span class="amount-value">{{ formatCurrency(row.amount) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="isPaidStatus(row.status) ? 'success' : 'warning'" size="small">
                  {{ row.status === "paid" ? "已支付" : "待支付" }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="创建时间" width="140">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.created_at_ms) }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { CheckCircle, CircleDollarSign, Clock, ListOrdered, ReceiptText } from "lucide-vue-next";
import { ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { PaymentOrder } from "@/api/types";
import { formatCurrency, formatTime, isPaidStatus } from "@/utils";

const orders = ref<PaymentOrder[]>([]);

const paidOrders = computed(() => orders.value.filter((item) => isPaidStatus(item.status)).length);
const pendingOrders = computed(() => orders.value.filter((item) => !isPaidStatus(item.status)).length);
const totalRevenue = computed(() => orders.value.filter((item) => isPaidStatus(item.status)).reduce((sum, item) => sum + item.amount, 0));

async function load() {
  try {
    orders.value = await adminAPI.orders();
  } catch (err) {
    console.error(err);
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.page-container {
  animation: fadeIn 0.4s ease-out;
}

.content-grid {
  display: grid;
  gap: 24px;
}

.stats-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 20px;
}

.stat-mini {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px !important;
}

.stat-mini-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-mini-icon.total {
  background: var(--primary-lighter);
  color: var(--primary-color);
}

.stat-mini-icon.paid {
  background: var(--success-light);
  color: var(--success-color);
}

.stat-mini-icon.pending {
  background: var(--warning-light);
  color: var(--warning-color);
}

.stat-mini-icon.revenue {
  background: var(--accent-light);
  color: var(--accent-color);
}

.stat-mini-content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-mini-label {
  font-size: 13px;
  color: var(--text-muted);
}

.stat-mini-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}

.orders-section {
  overflow: visible;
}

.order-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.modern-table {
  overflow: visible;
}

.trade-no {
  font-size: 13px;
  color: var(--text-secondary);
  background: var(--border-light);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
}

.user-id {
  font-weight: 600;
  color: var(--text-secondary);
}

.amount-value {
  font-size: 15px;
  font-weight: 700;
  color: var(--primary-color);
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
