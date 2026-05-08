<template>
  <el-dialog
    :model-value="visible"
    :title="account?.name || t('adminAccounts.accountDetails')"
    width="760px"
    destroy-on-close
    @close="$emit('close')"
  >
    <template v-if="account">
      <div class="detail-grid">
        <div class="surface-card detail-section">
          <div class="detail-section__header">{{ t('adminAccounts.basicInfo') }}</div>
          <div class="detail-list">
            <div v-for="row in basicRows" :key="row.key" class="detail-item">
              <span class="detail-label">{{ row.label }}</span>
              <code v-if="row.mono" class="detail-value mono">{{ row.value }}</code>
              <span v-else class="detail-value">{{ row.value }}</span>
            </div>
          </div>
        </div>

        <div class="surface-card detail-section">
          <div class="detail-section__header">{{ t('adminAccounts.modelScope') }}</div>
          <div v-if="modelScope.length" class="tag-list">
            <el-tag v-for="model in modelScope" :key="model" size="small" effect="plain">{{ model }}</el-tag>
          </div>
          <div v-else class="empty-text">{{ t('adminAccounts.noModelScope') }}</div>
        </div>

        <div class="surface-card detail-section">
          <div class="detail-section__header">{{ t('adminAccounts.credentialsSummary') }}</div>
          <div v-if="credentialSections.length" class="credential-sections">
            <div v-for="section in credentialSections" :key="section.key" class="credential-section">
              <div class="credential-section__title">{{ section.label }}</div>
              <div class="detail-list">
                <div v-for="row in section.rows" :key="row.key" class="detail-item">
                  <span class="detail-label">{{ row.label }}</span>
                  <code v-if="row.mono" class="detail-value mono">{{ row.value }}</code>
                  <span v-else class="detail-value">{{ row.value }}</span>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="empty-text">{{ t('adminAccounts.noCredentialSummary') }}</div>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ElDialog, ElTag } from "element-plus";
import { useI18n } from "vue-i18n";
import type { Account, AccountCredentialSummary } from "@/api/types";
import { formatTime } from "@/utils";

const props = defineProps<{
  visible: boolean;
  account: Account | null;
}>();

defineEmits<{
  close: [];
}>();

const { t } = useI18n();

function formatOptionalTime(value?: number) {
  return value ? formatTime(value) : t("common.noData");
}

function formatConcurrencyLimit(value?: number) {
  return value && value > 0 ? String(value) : t("adminAccounts.unlimited");
}

const modelScope = computed(() => {
  const raw = props.account?.model_scope_json || "[]";
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === "string" && item.trim() !== "") : [];
  } catch {
    return [];
  }
});

const basicRows = computed(() => {
  if (!props.account) return [];
  return [
    { key: "provider", label: t("adminAccounts.provider"), value: props.account.provider, mono: false },
    { key: "auth_type", label: t("adminAccounts.authType"), value: props.account.auth_type, mono: false },
    { key: "status", label: t("adminAccounts.status"), value: props.account.status, mono: false },
    { key: "base_url", label: t("adminAccounts.baseUrl"), value: props.account.base_url || t("common.noData"), mono: true },
    { key: "priority", label: t("adminAccounts.priority"), value: String(props.account.priority ?? 0), mono: false },
    { key: "concurrency_limit", label: t("adminAccounts.concurrencyLimit"), value: formatConcurrencyLimit(props.account.concurrency_limit), mono: false },
    { key: "expires_at_ms", label: t("adminAccounts.expireTime"), value: formatOptionalTime(props.account.expires_at_ms), mono: false },
    { key: "last_refreshed_at_ms", label: t("adminAccounts.lastRefreshedAt"), value: formatOptionalTime(props.account.last_refreshed_at_ms), mono: false },
  ];
});

const credentialSections = computed(() => {
  const cred = (props.account?.credentials || {}) as AccountCredentialSummary;
  const sections = [
    {
      key: "identity",
      label: t("adminAccounts.identitySummary"),
      rows: [
        { key: "email", label: t("adminAccounts.credentialEmail"), value: cred.email, mono: false },
        { key: "oauth_type", label: t("adminAccounts.credentialOAuthType"), value: cred.oauth_type, mono: false },
        { key: "project_id", label: t("adminAccounts.credentialProjectId"), value: cred.project_id, mono: true },
        { key: "organization_id", label: t("adminAccounts.credentialOrganizationId"), value: cred.organization_id, mono: true },
        { key: "account_id", label: t("adminAccounts.credentialAccountId"), value: cred.account_id, mono: true },
      ],
    },
    {
      key: "subscription",
      label: t("adminAccounts.subscriptionSummary"),
      rows: [
        { key: "plan_type", label: t("adminAccounts.credentialPlanType"), value: cred.plan_type, mono: false },
        { key: "subscription_until", label: t("adminAccounts.credentialSubscriptionUntil"), value: cred.subscription_until, mono: false },
        { key: "expires_at_ms", label: t("adminAccounts.expireTime"), value: cred.expires_at_ms ? formatTime(cred.expires_at_ms) : "", mono: false },
      ],
    },
    {
      key: "tokens",
      label: t("adminAccounts.tokenSummary"),
      rows: [
        { key: "api_key_masked", label: t("adminAccounts.credentialApiKey"), value: cred.api_key_masked, mono: true },
        { key: "refresh_token_masked", label: t("adminAccounts.credentialRefreshToken"), value: cred.refresh_token_masked, mono: true },
        { key: "access_token_masked", label: t("adminAccounts.credentialAccessToken"), value: cred.access_token_masked, mono: true },
        { key: "setup_token_masked", label: t("adminAccounts.credentialSetupToken"), value: cred.setup_token_masked, mono: true },
        { key: "token_url", label: t("adminAccounts.credentialTokenUrl"), value: cred.token_url, mono: true },
        { key: "redirect_uri", label: t("adminAccounts.credentialRedirectUri"), value: cred.redirect_uri, mono: true },
      ],
    },
  ];
  return sections
    .map((section) => ({
      ...section,
      rows: section.rows.filter((item) => item.value),
    }))
    .filter((section) => section.rows.length > 0);
});
</script>

<style scoped>
.detail-grid {
  display: grid;
  gap: 16px;
}

.detail-section {
  padding: 18px;
}

.detail-section__header {
  margin-bottom: 14px;
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.credential-sections {
  display: grid;
  gap: 16px;
}

.credential-section {
  padding: 14px;
  border-radius: var(--radius-lg);
  background: var(--bg-soft);
}

.credential-section__title {
  margin-bottom: 12px;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--text-muted);
  text-transform: uppercase;
}

.detail-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-item {
  display: grid;
  grid-template-columns: 160px 1fr;
  gap: 12px;
  align-items: start;
}

.detail-label {
  color: var(--text-muted);
  font-size: 13px;
}

.detail-value {
  color: var(--text-primary);
  word-break: break-all;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.empty-text {
  color: var(--text-muted);
  font-size: 13px;
}

@media (max-width: 640px) {
  .detail-item {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}
</style>
