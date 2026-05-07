<template>
  <div class="redeem-page">
    <div v-if="message" class="success-banner">
      <el-alert :title="message" type="success" :closable="false" show-icon />
    </div>
    <div v-if="error" class="error-banner">
      <el-alert :title="error" type="error" :closable="false" show-icon />
    </div>

    <div class="surface-card redeem-card">
      <div class="card-header">
        <h3 class="card-title">
          <Gift :size="20" />
          {{ t('redeem.redeemCode') }}
        </h3>
      </div>
      <div class="card-body">
        <p class="redeem-desc">{{ t('redeem.inputRedeemCode') }}</p>
        <el-form label-position="top" class="redeem-form">
          <el-form-item :label="t('redeem.redeemCode')">
            <el-input v-model="code" :placeholder="t('redeem.redeemCodePlaceholder')" size="large" :prefix-icon="Ticket" clearable @keyup.enter="submit" />
          </el-form-item>
          <el-button type="primary" size="large" class="redeem-btn" :loading="loading" @click="submit">
            <Gift :size="18" style="margin-right: 8px" />
            {{ t('redeem.redeemNow') }}
          </el-button>
        </el-form>

        <div class="help-section">
          <div class="help-title"><Info :size="14" /> {{ t('redeem.usageInstructions') }}</div>
          <ul class="help-list">
            <li><CheckCircle2 :size="13" />{{ t('redeem.instruction1') }}</li>
            <li><CheckCircle2 :size="13" />{{ t('redeem.instruction2') }}</li>
            <li><CheckCircle2 :size="13" />{{ t('redeem.instruction3') }}</li>
            <li><CheckCircle2 :size="13" />{{ t('redeem.instruction4') }}</li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { CheckCircle2, Gift, Info, Ticket } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput } from "element-plus";
import { userAPI } from "@/api/user";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const code = ref("");
const message = ref("");
const error = ref("");
const loading = ref(false);

async function submit() {
  if (!code.value.trim()) return;
  message.value = "";
  error.value = "";
  loading.value = true;
  try {
    await userAPI.redeem({ code: code.value });
    message.value = t('redeem.redeemSuccess');
    code.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('redeem.redeemFailed');
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.redeem-page {
  max-width: 480px;
  margin: 0 auto;
}

.redeem-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0 0 20px;
  line-height: 1.6;
}

.redeem-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.redeem-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
}

.help-section {
  margin-top: 28px;
  padding-top: 20px;
  border-top: 1px solid var(--border-light);
}

.help-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  margin-bottom: 12px;
}

.help-title svg {
  color: var(--text-muted);
}

.help-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.help-list li {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--text-muted);
  line-height: 1.4;
}

.help-list li svg {
  color: var(--primary-light);
  flex-shrink: 0;
}

.success-banner,
.error-banner {
  margin-bottom: 16px;
}
</style>
