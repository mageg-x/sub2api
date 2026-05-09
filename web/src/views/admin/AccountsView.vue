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
            <span class="stat-mini-value">{{ filteredAccounts.length }}</span>
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
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon warning">
            <CheckCircle :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">{{ t('adminAccounts.riskAccounts') }}</span>
            <span class="stat-mini-value">{{ riskAccounts }}</span>
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
          <div class="account-actions">
            <span class="account-count">{{ filteredAccounts.length }} / {{ accounts.length }} {{ t('adminAccounts.accountCount') }}</span>
            <span v-if="riskAccounts > 0" class="risk-hint">{{ t('adminAccounts.riskFirstHint') }}</span>
            <el-button type="primary" @click="showCreate = true">
              <Plus :size="16" style="margin-right: 6px" />
              {{ t('adminAccounts.createAccount') }}
            </el-button>
          </div>
        </div>
        <div class="card-body">
          <div class="quick-filters">
            <el-button :type="quickFilter === 'all' ? 'primary' : 'default'" size="small" @click="setQuickFilter('all')">
              {{ t('adminAccounts.quickAll') }}
            </el-button>
            <el-button :type="quickFilter === 'risk' ? 'warning' : 'default'" size="small" @click="setQuickFilter('risk')">
              {{ t('adminAccounts.quickRisk') }}
            </el-button>
            <el-button :type="quickFilter === 'expired' ? 'danger' : 'default'" size="small" @click="setQuickFilter('expired')">
              {{ t('adminAccounts.quickExpired') }}
            </el-button>
            <el-button :type="quickFilter === 'oauth' ? 'primary' : 'default'" size="small" @click="setQuickFilter('oauth')">
              {{ t('adminAccounts.quickOAuth') }}
            </el-button>
          </div>

          <div class="filter-toolbar">
            <el-input
              v-model="keyword"
              :placeholder="t('adminAccounts.searchPlaceholder')"
              clearable
              class="filter-item keyword-filter"
            />
            <el-select v-model="providerFilter" clearable class="filter-item" :placeholder="t('adminAccounts.providerPlaceholder')">
              <el-option :label="t('common.all')" value="" />
              <el-option v-for="provider in providerOptions" :key="provider" :label="provider" :value="provider" />
            </el-select>
            <el-select v-model="authTypeFilter" clearable class="filter-item" :placeholder="t('adminAccounts.authTypePlaceholder')">
              <el-option :label="t('common.all')" value="" />
              <el-option v-for="authType in authTypeOptions" :key="authType" :label="authType" :value="authType" />
            </el-select>
            <el-select v-model="statusFilter" clearable class="filter-item" :placeholder="t('adminAccounts.statusPlaceholder')">
              <el-option :label="t('common.all')" value="" />
              <el-option v-for="status in statusOptions" :key="status" :label="status" :value="status" />
            </el-select>
            <el-select v-model="healthFilter" clearable class="filter-item" :placeholder="t('adminAccounts.healthPlaceholder')">
              <el-option :label="t('common.all')" value="" />
              <el-option v-for="health in healthOptions" :key="health.value" :label="health.label" :value="health.value" />
            </el-select>
            <el-button class="filter-reset" @click="resetFilters">
              {{ t('common.reset') }}
            </el-button>
          </div>

          <el-table
            :data="filteredAccounts"
            :empty-text="filteredAccounts.length === 0 && hasActiveFilters ? t('adminAccounts.noFilteredAccounts') : t('adminAccounts.noAccounts')"
            class="modern-table"
            :stripe="true"
          >
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
            <el-table-column :label="t('adminAccounts.healthStatus')" width="110">
              <template #default="{ row }">
                <el-tag :type="accountHealth(row).tagType" size="small" effect="light">
                  {{ accountHealth(row).label }}
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
            <el-table-column :label="t('adminAccounts.actions')" width="160" fixed="right">
              <template #default="{ row }">
                <div class="table-actions">
                  <el-button
                    v-if="row.auth_type === 'oauth' && isUrgentRefresh(row)"
                    link
                    type="warning"
                    class="action-refresh-urgent"
                    :loading="refreshingId === row.id"
                    @click="handleRefresh(row.id)"
                  >
                    {{ t('adminAccounts.refreshNow') }}
                  </el-button>
                  <el-button link type="info" @click="openDetail(row)">
                    {{ t('adminAccounts.details') }}
                  </el-button>
                  <el-button link type="primary" @click="openEdit(row)">
                    {{ t('adminAccounts.edit') }}
                  </el-button>
                  <el-button
                    v-if="row.auth_type === 'oauth' && !isUrgentRefresh(row)"
                    link
                    type="primary"
                    :loading="refreshingId === row.id"
                    @click="handleRefresh(row.id)"
                  >
                    {{ t('adminAccounts.refresh') }}
                  </el-button>
                  <el-button link type="danger" :loading="deletingId === row.id" @click="handleDelete(row.id)">
                    {{ t('adminAccounts.delete') }}
                  </el-button>
                </div>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>

    <CreateAccountModal :visible="showCreate" @close="showCreate = false" @created="handleCreated" />
    <AccountDetailDialog :visible="showDetail" :account="selectedAccount" @close="showDetail = false" />
    <AccountEditDialog :visible="showEdit" :account="selectedAccount" @close="showEdit = false" @updated="handleUpdated" />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { Boxes, CheckCircle, KeyRound, Layers, Plus } from "lucide-vue-next";
import { ElAlert, ElButton, ElInput, ElMessage, ElMessageBox, ElOption, ElSelect, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Account } from "@/api/types";
import AccountDetailDialog from "@/components/admin/AccountDetailDialog.vue";
import AccountEditDialog from "@/components/admin/AccountEditDialog.vue";
import CreateAccountModal from "@/components/admin/CreateAccountModal.vue";
import { isActiveStatus } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const accounts = ref<Account[]>([]);
const error = ref("");
const showCreate = ref(false);
const showDetail = ref(false);
const showEdit = ref(false);
const selectedAccount = ref<Account | null>(null);
const keyword = ref("");
const providerFilter = ref("");
const authTypeFilter = ref("");
const statusFilter = ref("");
const healthFilter = ref("");
const quickFilter = ref<"all" | "risk" | "expired" | "oauth">("all");
const refreshingId = ref<number | null>(null);
const deletingId = ref<number | null>(null);

const providerOptions = computed(() => Array.from(new Set(accounts.value.map((item) => item.provider))).sort((a, b) => a.localeCompare(b)));
const authTypeOptions = computed(() => Array.from(new Set(accounts.value.map((item) => item.auth_type))).sort((a, b) => a.localeCompare(b)));
const statusOptions = computed(() => Array.from(new Set(accounts.value.map((item) => item.status))).sort((a, b) => a.localeCompare(b)));
const healthOptions = computed(() => [
  { value: "healthy", label: t("adminAccounts.healthHealthy") },
  { value: "expiring", label: t("adminAccounts.healthExpiring") },
  { value: "expired", label: t("adminAccounts.healthExpired") },
  { value: "stale", label: t("adminAccounts.healthNeedsRefresh") },
  { value: "inactive", label: t("adminAccounts.healthInactive") },
]);
const hasActiveFilters = computed(() =>
  Boolean(keyword.value.trim() || providerFilter.value || authTypeFilter.value || statusFilter.value || healthFilter.value || quickFilter.value !== "all"),
);

function accountHealth(item: Account) {
  const now = Date.now();
  const expiresAt = item.expires_at_ms || item.credentials?.expires_at_ms || 0;
  const refreshThreshold = 3 * 24 * 60 * 60 * 1000;

  if (!isActiveStatus(item.status)) {
    return { key: "inactive", label: t("adminAccounts.healthInactive"), tagType: "info" as const };
  }
  if (expiresAt > 0 && expiresAt <= now) {
    return { key: "expired", label: t("adminAccounts.healthExpired"), tagType: "danger" as const };
  }
  if (item.auth_type === "oauth" && expiresAt > 0 && expiresAt - now <= refreshThreshold) {
    return { key: "expiring", label: t("adminAccounts.healthExpiring"), tagType: "warning" as const };
  }
  if (item.auth_type === "oauth" && !item.last_refreshed_at_ms) {
    return { key: "stale", label: t("adminAccounts.healthNeedsRefresh"), tagType: "warning" as const };
  }
  return { key: "healthy", label: t("adminAccounts.healthHealthy"), tagType: "success" as const };
}

function healthPriority(item: Account) {
  const key = accountHealth(item).key;
  if (key === "expired") return 0;
  if (key === "expiring") return 1;
  if (key === "stale") return 2;
  if (key === "healthy") return 3;
  return 4;
}

function isUrgentRefresh(item: Account) {
  const key = accountHealth(item).key;
  return item.auth_type === "oauth" && (key === "expired" || key === "expiring" || key === "stale");
}

const filteredAccounts = computed(() => {
  const query = keyword.value.trim().toLowerCase();
  return accounts.value
    .filter((item) => {
      if (quickFilter.value === "risk" && !["expired", "expiring", "stale"].includes(accountHealth(item).key)) return false;
      if (quickFilter.value === "expired" && accountHealth(item).key !== "expired") return false;
      if (quickFilter.value === "oauth" && item.auth_type !== "oauth") return false;
      if (providerFilter.value && item.provider !== providerFilter.value) return false;
      if (authTypeFilter.value && item.auth_type !== authTypeFilter.value) return false;
      if (statusFilter.value && item.status !== statusFilter.value) return false;
      if (healthFilter.value && accountHealth(item).key !== healthFilter.value) return false;
      if (!query) return true;

      const haystacks = [
        item.provider,
        item.name,
        item.auth_type,
        item.status,
        item.base_url,
        item.credentials?.email,
        item.credentials?.organization_id,
        item.credentials?.project_id,
        item.credentials?.account_id,
        accountHealth(item).label,
      ];

      return haystacks.some((value) => String(value || "").toLowerCase().includes(query));
    })
    .slice()
    .sort((a, b) => {
      const healthDiff = healthPriority(a) - healthPriority(b);
      if (healthDiff !== 0) return healthDiff;
      if (a.priority !== b.priority) return b.priority - a.priority;
      return a.id - b.id;
    });
});
const activeAccounts = computed(() => filteredAccounts.value.filter((item) => isActiveStatus(item.status)).length);
const providerCount = computed(() => new Set(filteredAccounts.value.map((item) => item.provider)).size);
const riskAccounts = computed(() =>
  filteredAccounts.value.filter((item) => {
    const health = accountHealth(item).key;
    return health === "expired" || health === "expiring" || health === "stale";
  }).length,
);

async function load() {
  try {
    error.value = "";
    accounts.value = await adminAPI.accounts();
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('adminAccounts.loadFailed');
  }
}

onMounted(() => {
  void load();
});

async function handleCreated() {
  showCreate.value = false;
  await load();
}

async function handleUpdated() {
  showEdit.value = false;
  await load();
}

function openDetail(account: Account) {
  selectedAccount.value = account;
  showDetail.value = true;
}

function openEdit(account: Account) {
  selectedAccount.value = account;
  showEdit.value = true;
}

function resetFilters() {
  keyword.value = "";
  providerFilter.value = "";
  authTypeFilter.value = "";
  statusFilter.value = "";
  healthFilter.value = "";
  quickFilter.value = "all";
}

function setQuickFilter(value: "all" | "risk" | "expired" | "oauth") {
  quickFilter.value = value;
}

async function handleRefresh(id: number) {
  if (refreshingId.value === id) return;
  refreshingId.value = id;
  try {
    await adminAPI.refreshAccount(id);
    ElMessage.success(t('adminAccounts.refreshSuccess'));
    await load();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t('adminAccounts.refreshFailed'));
  } finally {
    refreshingId.value = null;
  }
}

async function handleDelete(id: number) {
  if (deletingId.value === id) return;
  try {
    await ElMessageBox.confirm(t('adminAccounts.deleteConfirm'), t('common.warning'), {
      type: 'warning',
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
    });
    deletingId.value = id;
    await adminAPI.deleteAccount(id);
    ElMessage.success(t('adminAccounts.deleteSuccess'));
    await load();
  } catch (err) {
    if (err === 'cancel' || err === 'close') return;
    ElMessage.error(err instanceof Error ? err.message : t('adminAccounts.deleteFailed'));
  } finally {
    deletingId.value = null;
  }
}
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
  grid-template-columns: repeat(4, 1fr);
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

.stat-mini-icon.warning {
  background: var(--warning-light);
  color: var(--warning-color);
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

.account-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.risk-hint {
  font-size: 12px;
  color: var(--warning-color);
  background: var(--warning-light);
  padding: 6px 10px;
  border-radius: var(--radius-full);
  font-weight: 600;
}

.filter-toolbar {
  display: grid;
  grid-template-columns: minmax(220px, 1.6fr) repeat(4, minmax(140px, 1fr)) auto;
  gap: 12px;
  margin-bottom: 16px;
}

.quick-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}

.filter-item {
  width: 100%;
}

.keyword-filter {
  min-width: 0;
}

.filter-reset {
  min-width: 92px;
}

.modern-table {
  overflow: visible;
}

.provider-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.table-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.action-refresh-urgent {
  font-weight: 700;
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

@media (max-width: 640px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }

  .filter-toolbar {
    grid-template-columns: 1fr;
  }

  .account-actions {
    width: 100%;
    justify-content: space-between;
  }
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
