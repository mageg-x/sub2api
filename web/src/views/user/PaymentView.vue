<template>
  <div class="page-container">
    <div class="content-grid">
      <div class="stats-row">
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon balance">
            <WalletCards :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('payment.currentBalance') }}</span>
            <span class="stat-mini-value">{{ formatCurrency(session.user?.balance || 0) }} {{ t('common.currency') }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon orders">
            <ReceiptText :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('payment.totalOrders') }}</span>
            <span class="stat-mini-value">{{ orders.length }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon paid">
            <CheckCircle :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('payment.paidOrders') }}</span>
            <span class="stat-mini-value">{{ paidOrders }}</span>
          </div>
        </div>
      </div>

      <div v-if="error" class="surface-card error-section">
        <el-alert :title="error" type="error" :closable="false" show-icon />
      </div>

      <div class="main-grid">
        <div class="surface-card create-order-section">
          <div class="card-header">
            <h3 class="card-title">
              <CreditCard :size="20" />
              {{ t('payment.initiatePayment') }}
            </h3>
          </div>
          <div class="card-body">
            <el-form label-position="top" class="payment-form">
              <el-form-item :label="t('payment.rechargeAmount')" class="form-item-highlight">
                <el-input v-model.number="form.amount" type="number" size="large" :placeholder="t('payment.pleaseInputAmount')">
                  <template #suffix>
                    <span class="input-suffix">{{ t('common.currency') }}</span>
                  </template>
                </el-input>
                <div class="quick-amounts">
                  <el-button v-for="amount in quickAmounts" :key="amount" size="small" @click="form.amount = amount"> {{ amount }} {{ t('common.currency') }} </el-button>
                </div>
              </el-form-item>

              <el-form-item :label="t('payment.orderTitle')" class="form-item">
                <el-input v-model="form.subject" :placeholder="t('payment.orderTitlePlaceholder')" size="large" />
              </el-form-item>

              <el-button type="primary" size="large" :loading="submitting" class="submit-button" @click="createOrder">
                <CreditCard :size="18" />
                {{ t('payment.createPaymentOrder') }}
              </el-button>
            </el-form>
          </div>
        </div>

        <div class="surface-card result-section">
          <div class="card-header">
            <h3 class="card-title">
              <ReceiptText :size="20" />
              {{ t('payment.paymentReceipt') }}
            </h3>
          </div>
          <div class="card-body">
            <div v-if="!result" class="empty-state-inline">
              <div class="empty-icon-inline">
                <ReceiptText :size="24" />
              </div>
              <p>{{ t('payment.receiptInfo') }}</p>
            </div>
            <div v-else class="result-display">
              <pre class="json-preview mono">{{ prettyJSON(result) }}</pre>
              <p class="helper-text">{{ t('payment.orderStatusNote') }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="surface-card orders-section">
        <div class="card-header">
          <h3 class="card-title">
            <ListOrdered :size="20" />
            {{ t('payment.myOrders') }}
          </h3>
          <span class="order-count">{{ orders.length }} {{ t('payment.orderCount') }}</span>
        </div>
        <div class="card-body">
          <el-table :data="orders" :empty-text="t('common.noOrders')" class="modern-table" :stripe="true">
            <el-table-column prop="out_trade_no" :label="t('payment.merchantOrderNo')" min-width="160">
              <template #default="{ row }">
                <code class="trade-no mono">{{ row.out_trade_no }}</code>
              </template>
            </el-table-column>
            <el-table-column :label="t('payment.amount')" width="90">
              <template #default="{ row }">
                <span class="amount-value">{{ formatCurrency(row.amount) }}</span>
              </template>
            </el-table-column>
            <el-table-column :label="t('payment.status')" width="80">
              <template #default="{ row }">
                <el-tag :type="isPaidStatus(row.status) ? 'success' : 'warning'">
                  {{ row.status === "paid" ? t('payment.paid') : t('payment.pending') }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('payment.createTime')" width="140">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.created_at_ms) }}</span>
              </template>
            </el-table-column>
            <el-table-column :label="t('payment.actions')" width="70">
              <template #default="{ row }">
                <el-link type="primary" @click="goDetail(row.id)"> {{ t('payment.viewDetails') }} </el-link>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { CheckCircle, CreditCard, ListOrdered, ReceiptText, WalletCards } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput, ElLink, ElTable, ElTableColumn, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import { session } from "@/store/session";
import type { PaymentCreateResponse, PaymentOrder } from "@/api/types";
import { formatCurrency, formatTime, isPaidStatus, prettyJSON } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const router = useRouter();
const result = ref<PaymentCreateResponse | null>(null);
const orders = ref<PaymentOrder[]>([]);
const error = ref("");
const submitting = ref(false);
const form = reactive({
  amount: 100,
  subject: "Balance Recharge",
});

const quickAmounts = [50, 100, 200, 500, 1000];

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
    error.value = err instanceof Error ? err.message : t('payment.createFailed');
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

.stat-mini-icon.balance {
  background: var(--primary-lighter);
  color: var(--primary-color);
}

.stat-mini-icon.orders {
  background: var(--info-light);
  color: var(--info-color);
}

.stat-mini-icon.paid {
  background: var(--success-light);
  color: var(--success-color);
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

.error-section {
  padding: 0;
}

.main-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.create-order-section,
.result-section {
  overflow: visible;
}

.payment-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-item-highlight {
  margin-bottom: 16px;
}

.form-item {
  margin-bottom: 16px;
}

.input-suffix {
  color: var(--text-muted);
  font-weight: 500;
}

.quick-amounts {
  display: flex;
  gap: 10px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.quick-amounts .el-button {
  min-width: 80px;
}

.submit-button {
  width: 100%;
  height: 52px !important;
  font-size: 16px !important;
  margin-top: 8px;
}

.empty-state-inline {
  text-align: center;
  padding: 48px 20px;
}

.empty-icon-inline {
  width: 64px;
  height: 64px;
  background: var(--border-light);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
  color: var(--text-muted);
}

.empty-state-inline p {
  color: var(--text-muted);
  font-size: 14px;
  margin: 0;
}

.result-display {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.json-preview {
  background: var(--border-light);
  border-radius: var(--radius-lg);
  padding: 20px;
  font-size: 13px;
  color: var(--text-secondary);
  overflow-x: auto;
  margin: 0;
  line-height: 1.8;
}

.helper-text {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0;
  line-height: 1.7;
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
