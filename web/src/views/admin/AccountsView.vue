<template>
  <div class="page-container">
    <div class="content-grid">
      <div class="stats-row">
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon total">
            <Boxes :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('adminAccounts.totalAccounts') }}</span>
            <span class="stat-mini-value">{{ accounts.length }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon active">
            <CheckCircle :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('adminAccounts.activeAccounts') }}</span>
            <span class="stat-mini-value">{{ activeAccounts }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon providers">
            <Layers :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('adminAccounts.providerCount') }}</span>
            <span class="stat-mini-value">{{ providerCount }}</span>
          </div>
        </div>
      </div>

      <div v-if="error" class="surface-card error-banner">
        <el-alert :title="error" type="error" :closable="false" show-icon />
      </div>

      <div class="surface-card accounts-section">
        <div class="card-header">
          <h3 class="card-title">
            <Boxes :size="20" />
            {{ t('adminAccounts.accountList') }}
          </h3>
          <span class="account-count">{{ accounts.length }} {{ t('adminAccounts.accountCount') }}</span>
        </div>
        <div class="card-body">
          <el-table :data="accounts" :empty-text="t('adminAccounts.noAccounts')" class="modern-table" :stripe="true">
            <el-table-column prop="provider" :label="t('adminAccounts.provider')" width="120">
              <template #default="{ row }">
                <div class="provider-cell">
                  <div class="provider-badge" :class="row.provider.toLowerCase()">
                    {{ row.provider.charAt(0) }}
                  </div>
                  <span>{{ row.provider }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="name" :label="t('adminAccounts.accountName')" min-width="130">
              <template #default="{ row }">
                <div class="name-cell">
                  <KeyRound :size="14" />
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="auth_type" :label="t('adminAccounts.authType')" width="100">
              <template #default="{ row }">
                <el-tag :type="row.auth_type === 'oauth' ? 'primary' : 'info'" size="small">
                  {{ row.auth_type }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('adminAccounts.status')" width="80">
              <template #default="{ row }">
                <el-tag :type="isActiveStatus(row.status) ? 'success' : 'info'" size="small">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('adminAccounts.priority')" width="70">
              <template #default="{ row }">
                <span class="priority-value">{{ row.priority || 0 }}</span>
              </template>
            </el-table-column>
            <el-table-column :label="t('adminAccounts.concurrencyLimit')" width="80">
              <template #default="{ row }">
                <span class="limit-value">{{ row.concurrency_limit || "∞" }}</span>
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
import { Boxes, CheckCircle, KeyRound, Layers } from "lucide-vue-next";
import { ElAlert, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Account } from "@/api/types";
import { isActiveStatus } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const accounts = ref<Account[]>([]);
const error = ref("");

const activeAccounts = computed(() => accounts.value.filter((item) => isActiveStatus(item.status)).length);
const providerCount = computed(() => new Set(accounts.value.map((item) => item.provider)).size);

async function load() {
  try {
    accounts.value = await adminAPI.accounts();
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('adminAccounts.loadFailed');
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
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.stat-mini {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px !important;
}

.stat-mini-icon {
  width: 34px;
  height: 34px;
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

.stat-mini-icon.providers {
  background: var(--info-light);
  color: var(--info-color);
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

.error-banner {
  padding: 0;
}

.accounts-section {
  overflow: visible;
}

.account-count {
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

.provider-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.provider-badge {
  width: 28px;
  height: 28px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 14px;
  color: white;
}

.provider-badge.openai {
  background: linear-gradient(135deg, #10a37f, #10b981);
}

.provider-badge.claude {
  background: linear-gradient(135deg, #d4a574, #e8c49a);
}

.provider-badge.gemini {
  background: linear-gradient(135deg, #4285f4, #667eea);
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.name-cell svg {
  color: var(--text-muted);
}

.priority-value,
.limit-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  font-family: "SF Mono", "Monaco", monospace;
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
