<template>
  <div class="order-detail-page">
    <div v-if="loading" class="loading-state">
      <div class="loading-spinner"></div>
      <span>{{ t('common.loading') }}</span>
    </div>
    <div v-else-if="error" class="surface-card error-section">
      <el-alert :title="error" type="error" :closable="false" show-icon />
    </div>
    <template v-else-if="order">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <ReceiptText :size="20" />
            {{ t('orderDetail.orderDetails') }}
          </h3>
          <el-tag :type="order.status === 'paid' ? 'success' : 'warning'" size="small">
            {{ order.status === 'paid' ? t('dashboard.paid') : t('dashboard.pending') }}
          </el-tag>
        </div>
        <div class="card-body">
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-label">{{ t('orderDetail.merchantOrderNo') }}</span>
              <div class="detail-value-row">
                <code class="detail-value mono">{{ order.out_trade_no }}</code>
                <el-button text size="small" @click="copyText(order.out_trade_no)">
                  <Copy :size="14" />
                </el-button>
              </div>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('orderDetail.rechargeAmount') }}</span>
              <span class="detail-value amount">{{ formatCurrency(order.amount) }} {{ t('common.currency') }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('orderDetail.creditedAmount') }}</span>
              <span class="detail-value">{{ formatCurrency(order.credited_amount) }} {{ t('common.currency') }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('orderDetail.paymentChannel') }}</span>
              <el-tag size="small">{{ order.provider }}</el-tag>
            </div>
            <div class="detail-item">
              <span class="detail-label">{{ t('orderDetail.createTime') }}</span>
              <span class="detail-value">{{ formatTime(order.created_at_ms) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <Lightbulb :size="20" />
            {{ t('orderDetail.faq') }}
          </h3>
        </div>
        <div class="card-body">
          <ul class="tips-list">
            <li>{{ t('orderDetail.faq1') }}</li>
            <li>{{ t('orderDetail.faq2') }}</li>
            <li>{{ t('orderDetail.faq3') }}</li>
          </ul>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Copy, Lightbulb, ReceiptText } from "lucide-vue-next";
import { ElAlert, ElButton, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import type { PaymentOrder } from "@/api/types";
import { formatCurrency, formatTime, copyToClipboard } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const order = ref<PaymentOrder | null>(null);
const loading = ref(true);
const error = ref("");

function copyText(value: string) {
  void copyToClipboard(value);
}

async function load() {
  const id = Number(route.params.id);
  if (!id || Number.isNaN(id)) {
    router.replace("/user/payment");
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    order.value = await userAPI.orderByID(id);
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('common.loadingFailed');
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.order-detail-page {
  max-width: 720px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.loading-state {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-muted);
  font-size: 14px;
  padding: 40px;
  justify-content: center;
}

.loading-spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border-default);
  border-top-color: var(--primary-color);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px 32px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.detail-label {
  font-size: 12px;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.detail-value-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-value {
  font-size: 15px;
  color: var(--text-primary);
  font-weight: 500;
}

.detail-value.mono {
  font-size: 13px;
  background: var(--border-light);
  padding: 6px 10px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

.detail-value.amount {
  font-size: 20px;
  font-weight: 700;
  color: var(--warning-color);
  letter-spacing: -0.02em;
}

.tips-list {
  margin: 0;
  padding-left: 20px;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 2.2;
}

.tips-list li::marker {
  color: var(--primary-color);
}

@media (max-width: 600px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
