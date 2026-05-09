<template>
  <div>
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <div class="surface-card stat-mini">
        <div class="stat-mini-icon total">
          <Ticket :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">{{ t('adminCoupons.totalCoupons') }}</span>
          <span class="stat-mini-value">{{ items.length }}</span>
        </div>
      </div>

      <div class="surface-card stat-mini">
        <div class="stat-mini-icon active">
          <CheckCircle :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">{{ t('adminCoupons.activated') }}</span>
          <span class="stat-mini-value">{{ items.filter((item) => item.status === "active").length }}</span>
        </div>
      </div>

      <div class="surface-card stat-mini">
        <div class="stat-mini-icon used">
          <Gift :size="20" />
        </div>
        <div class="stat-mini-content">
          <span class="stat-mini-label">{{ t('adminCoupons.redeemed') }}</span>
          <span class="stat-mini-value">{{ items.reduce((sum, item) => sum + item.used_count, 0) }}</span>
        </div>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Ticket :size="20" />
          {{ t('adminCoupons.createCoupon') }}
        </h3>
      </div>
      <div class="card-body">
        <el-form label-position="top" class="modern-form">
          <div class="form-grid">
            <el-form-item :label="t('adminCoupons.couponCode')">
              <el-input v-model="form.code" :placeholder="t('adminCoupons.leaveBlankAutoGenerate')">
                <template #prefix><Ticket :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminCoupons.amount')">
              <el-input v-model.number="form.amount" type="number" :placeholder="t('adminCoupons.rechargeAmountInCents')">
                <template #prefix><CircleDollarSign :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminCoupons.kind')">
              <el-select v-model="form.kind">
                <el-option :label="t('adminCoupons.kindBalance')" value="balance" />
                <el-option :label="t('adminCoupons.kindPercent')" value="percent" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('adminCoupons.maxUses')">
              <el-input v-model.number="form.max_uses" type="number" :placeholder="t('adminCoupons.defaultOne')">
                <template #prefix><Hash :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item :label="t('adminCoupons.expirationTime')">
              <el-date-picker
                v-model="expiresAtDate"
                type="datetime"
                :placeholder="t('adminCoupons.zeroNoExpiry')"
                clearable
                style="width: 100%"
              />
            </el-form-item>
          </div>
          <el-button type="primary" @click="create">
            <Ticket :size="16" style="margin-right: 6px" />
            {{ t('adminCoupons.createCouponBtn') }}
          </el-button>
        </el-form>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Gift :size="20" />
          {{ t('adminCoupons.couponList') }}
        </h3>
        <span class="coupon-count">{{ items.length }} {{ t('adminCoupons.couponCountLabel') }}</span>
      </div>
      <div class="card-body">
        <el-table :data="items" :empty-text="t('adminCoupons.noCoupons')" class="modern-table" :stripe="true">
          <el-table-column prop="code" :label="t('adminCoupons.couponCode')" min-width="160">
            <template #default="{ row }">
              <code class="code-value mono">{{ row.code }}</code>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminCoupons.amount')" width="100">
            <template #default="{ row }">
              <span class="amount-value">
                {{ row.kind === 'percent' ? `${row.amount}%` : `${formatCurrency(row.amount)} ${t('common.currency')}` }}
              </span>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminCoupons.kind')" width="100">
            <template #default="{ row }">
              <el-tag size="small" :type="row.kind === 'percent' ? 'warning' : 'success'">
                {{ row.kind === 'percent' ? t('adminCoupons.kindPercent') : t('adminCoupons.kindBalance') }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminCoupons.used')" width="80">
            <template #default="{ row }">
              <span class="used-count">{{ row.used_count }} / {{ row.max_uses }}</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('common.status')" width="90">
            <template #default="{ row }">
              <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('adminCoupons.expirationTime')" width="150">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(Number(row.expires_at_ms || 0)) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CheckCircle, CircleDollarSign, Clock, Gift, Hash, Ticket } from "lucide-vue-next";
import { ElButton, ElDatePicker, ElForm, ElFormItem, ElInput, ElMessage, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Coupon } from "@/api/types";
import { formatCurrency, formatTime } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const items = ref<Coupon[]>([]);
const form = reactive({
  code: "",
  kind: "balance",
  amount: 10000,
  max_uses: 1,
  expires_at_ms: 0,
});

const expiresAtDate = computed({
  get: () => form.expires_at_ms ? new Date(form.expires_at_ms) : null,
  set: (val: Date | null) => { form.expires_at_ms = val ? val.getTime() : 0; }
});

async function load() {
  try {
    items.value = await adminAPI.coupons();
  } catch {
    items.value = [];
  }
}

async function create() {
  if (!form.amount || form.amount <= 0) return;
  try {
    await adminAPI.createCoupon({ ...form });
    form.code = "";
    await load();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t('common.operationFailed'));
  }
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
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

.stat-mini-icon.active {
  background: var(--success-light);
  color: var(--success-color);
}

.stat-mini-icon.used {
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

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 20px;
}

.coupon-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.code-value {
  font-size: 13px;
  background: var(--border-light);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
}

.amount-value {
  font-size: 14px;
  font-weight: 700;
  color: var(--success-color);
}

.used-count {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.time-text {
  font-size: 13px;
  color: var(--text-muted);
}
</style>
