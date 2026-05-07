<template>
  <div>
    <div class="card-grid">
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <WalletCards :size="24" />
          </div>
        </div>
        <p class="stat-label">账户余额</p>
        <p class="stat-value">{{ formatCurrency(session.user?.balance || 0) }}</p>
        <p class="stat-helper">单位：元</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <ReceiptText :size="24" />
          </div>
        </div>
        <p class="stat-label">最近请求</p>
        <p class="stat-value">{{ usage.length }}</p>
        <p class="stat-helper">最近加载的调用记录</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <ListOrdered :size="24" />
          </div>
        </div>
        <p class="stat-label">订单数量</p>
        <p class="stat-value">{{ orders.length }}</p>
        <p class="stat-helper">充值与回调状态追踪</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <ShieldUser :size="24" />
          </div>
        </div>
        <p class="stat-label">用户角色</p>
        <p class="stat-value">{{ session.user?.role || "-" }}</p>
        <p class="stat-helper">当前登录身份</p>
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
              <el-button size="large" @click="router.push('/user/access-guide')">
                <BookOpenText :size="18" />
                接入指南
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <ShieldCheck :size="20" />
          功能概览
        </h3>
      </div>
      <div class="card-body">
        <p class="helper-copy">当前用户端已覆盖登录、API Keys 管理、用量查看、余额充值、订单追踪、兑换码兑换、公告查看以及个人资料维护。 通过左侧导航栏可以快速访问各项功能。</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { Bell, Bolt, BookOpenText, KeyRound, ListOrdered, ReceiptText, ShieldCheck, ShieldUser, WalletCards } from "lucide-vue-next";
import { ElButton, ElTag } from "element-plus";
import { session } from "@/store/session";
import { userAPI } from "@/api/user";
import type { PaymentOrder, UsageLog } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";

const router = useRouter();

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

<style scoped>
.dashboard-grid {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 24px;
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
  background: var(--border-color);
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
  height: 48px;
  padding: 0 20px;
}

.quick-actions .el-button :deep(svg) {
  margin-right: 10px;
}

.helper-copy {
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.8;
  margin: 0;
}

@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>
