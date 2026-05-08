<template>
  <el-dialog
    :model-value="visible"
    :title="t('adminAccounts.createAccount')"
    width="880px"
    destroy-on-close
    class="create-account-dialog"
    @close="handleClose"
  >
    <div class="wizard-shell">
      <div class="wizard-intro">
        <div>
          <h3 class="wizard-title">{{ t("adminAccounts.accountWizardTitle") }}</h3>
          <p class="wizard-subtitle">{{ t("adminAccounts.accountWizardSubtitle") }}</p>
        </div>
        <div class="wizard-provider-chip">
          <span class="wizard-provider-chip__dot" :class="form.provider"></span>
          <span>{{ currentProvider?.label || "-" }}</span>
        </div>
      </div>

      <div v-if="currentProvider" class="provider-notice">
        <strong>{{ currentProvider.label }}</strong>
        <span>{{ currentProvider.notice }}</span>
      </div>

      <el-form v-if="currentProvider" label-position="top" class="modern-form">
        <div class="form-grid form-grid--three">
          <el-form-item :label="t('adminAccounts.provider')">
            <el-select v-model="form.provider">
              <el-option v-for="item in providers" :key="item.name" :label="item.label" :value="item.name" />
            </el-select>
          </el-form-item>

          <el-form-item :label="t('adminAccounts.authType')">
            <el-select v-model="form.authMode">
              <el-option v-for="item in authModeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>

          <el-form-item :label="t('adminAccounts.accountName')">
            <el-input v-model="form.name" :placeholder="t('adminAccounts.accountNamePlaceholder')" />
          </el-form-item>
        </div>

        <div class="form-grid form-grid--three">
          <el-form-item :label="t('adminAccounts.baseUrl')">
            <el-input v-model="form.baseUrl" :placeholder="currentProvider.base_url_placeholder" />
          </el-form-item>

          <el-form-item :label="t('adminAccounts.priority')">
            <el-input-number v-model="form.priority" :min="1" :max="10000" class="full-width" />
          </el-form-item>

          <el-form-item :label="t('adminAccounts.concurrencyLimit')">
            <el-input-number v-model="form.concurrencyLimit" :min="0" :max="128" class="full-width" />
            <div class="field-hint">
              {{ form.concurrencyLimit === 0 ? t('adminAccounts.unlimitedHint') : t('adminAccounts.concurrencyLimitCreateHint') }}
            </div>
          </el-form-item>
        </div>

        <div v-if="accountFields.length" class="form-grid form-grid--three">
          <template v-for="field in accountFields" :key="field.key">
            <el-form-item v-if="isFieldVisible(field)" :label="field.label">
              <component :is="fieldComponent(field)" v-model="fieldValues[field.key]" v-bind="fieldProps(field)">
                <el-option
                  v-for="item in field.options || []"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </component>
              <div v-if="field.help" class="field-hint">{{ field.help }}</div>
            </el-form-item>
          </template>
        </div>

        <el-form-item :label="t('adminAccounts.modelScope')">
          <el-input
            v-model="form.modelScopeText"
            type="textarea"
            :rows="3"
            :placeholder="t('adminAccounts.modelScopePlaceholder')"
          />
        </el-form-item>

        <div v-if="isOAuthMode" class="oauth-panel">
          <div class="oauth-panel__header">
            <div>
              <h4>{{ t("adminAccounts.oauthFlow") }}</h4>
              <p>{{ t("adminAccounts.oauthFlowHint") }}</p>
            </div>
            <el-button :loading="oauthStarting" type="primary" @click="startOAuth">
              {{ t("adminAccounts.generateAuthUrl") }}
            </el-button>
          </div>

          <div v-if="oauthFields.length" class="form-grid form-grid--two">
            <template v-for="field in oauthFields" :key="field.key">
              <el-form-item v-if="isFieldVisible(field)" :label="field.label">
                <component :is="fieldComponent(field)" v-model="fieldValues[field.key]" v-bind="fieldProps(field)">
                  <el-option
                    v-for="item in field.options || []"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </component>
                <div v-if="field.help" class="field-hint">{{ field.help }}</div>
              </el-form-item>
            </template>
          </div>

          <div v-if="oauth.authUrl" class="oauth-result">
            <el-alert :title="t('adminAccounts.oauthGenerated')" type="success" :closable="false" show-icon />
            <el-form-item :label="t('adminAccounts.authUrl')">
              <el-input :model-value="oauth.authUrl" readonly>
                <template #append>
                  <el-button @click="copyText(oauth.authUrl)">{{ t("common.copy") }}</el-button>
                </template>
              </el-input>
            </el-form-item>
            <div class="oauth-state-row">
              <span>{{ t("adminAccounts.oauthState") }}: <code class="mono">{{ oauth.state || "-" }}</code></span>
              <span>{{ t("adminAccounts.oauthSession") }}: <code class="mono">{{ oauth.sessionId || "-" }}</code></span>
            </div>
          </div>

          <div class="form-grid form-grid--two">
            <el-form-item :label="t('adminAccounts.oauthStateInput')">
              <el-input v-model="oauth.stateInput" :placeholder="t('adminAccounts.oauthStatePlaceholder')" />
            </el-form-item>

            <el-form-item :label="t('adminAccounts.oauthCode')">
              <el-input v-model="oauth.code" :placeholder="t('adminAccounts.oauthCodePlaceholder')" />
            </el-form-item>
          </div>
        </div>

        <div v-else class="apikey-panel">
          <el-form-item :label="apiKeyField.label">
            <el-input
              v-model="form.apiKey"
              type="textarea"
              :rows="4"
              :placeholder="apiKeyField.placeholder || t('adminAccounts.apiKeyPlaceholder')"
              show-password
            />
          </el-form-item>
        </div>
      </el-form>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">{{ t("common.cancel") }}</el-button>
        <el-button v-if="isOAuthMode" :loading="submitting" type="primary" @click="completeOAuthCreate">
          {{ t("adminAccounts.completeAuthorization") }}
        </el-button>
        <el-button v-else :loading="submitting" type="primary" @click="createAPIKeyAccount">
          {{ t("adminAccounts.createAccount") }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { ElAlert, ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElMessage, ElOption, ElSelect } from "element-plus";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api/admin";
import type { AccountCredentials, ProviderCapability, ProviderCapabilityField } from "@/api/types";

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  close: [];
  created: [];
}>();

const { t } = useI18n();

const state = reactive({
  loadingCapabilities: false,
  oauthStarting: false,
  submitting: false,
});

const providers = reactive<ProviderCapability[]>([]);

const form = reactive({
  provider: "",
  authMode: "oauth",
  name: "",
  baseUrl: "",
  priority: 100,
  concurrencyLimit: 4,
  modelScopeText: "",
  apiKey: "",
});

const fieldValues = reactive<Record<string, string>>({});

const oauth = reactive({
  authUrl: "",
  sessionId: "",
  state: "",
  stateInput: "",
  code: "",
});

const currentProvider = computed(() => providers.find((item) => item.name === form.provider) || null);
const isOAuthMode = computed(() => form.authMode === "oauth");
const authModeOptions = computed(() => currentProvider.value?.auth_modes || []);
const apiKeyField = computed<ProviderCapabilityField>(() => currentProvider.value?.api_key_field || {
  key: "api_key",
  label: t("adminAccounts.apiKey"),
  type: "textarea",
  required: true,
});
const accountFields = computed(() => currentProvider.value?.account_fields || []);
const oauthFields = computed(() => currentProvider.value?.oauth_fields || []);
const oauthStarting = computed(() => state.oauthStarting);
const submitting = computed(() => state.submitting);

watch(
  () => props.visible,
  async (visible) => {
    if (!visible) return;
    await ensureCapabilities();
    resetForm();
  },
);

watch(
  () => form.provider,
  (providerName) => {
    const capability = providers.find((item) => item.name === providerName);
    if (!capability) return;
    form.authMode = capability.default_auth_mode || capability.auth_modes[0]?.value || "oauth";
    form.baseUrl = capability.default_base_url || "";
    applyFieldDefaults(capability);
    resetOAuthState();
  },
);

function applyFieldDefaults(capability: ProviderCapability) {
  const fields = [...(capability.account_fields || []), ...(capability.oauth_fields || [])];
  for (const field of fields) {
    fieldValues[field.key] = field.default_value || "";
  }
}

async function ensureCapabilities() {
  if (providers.length > 0 || state.loadingCapabilities) return;
  state.loadingCapabilities = true;
  try {
    const items = await adminAPI.providerCapabilities();
    providers.splice(0, providers.length, ...items);
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t("common.loadFailed"));
  } finally {
    state.loadingCapabilities = false;
  }
}

function resetOAuthState() {
  oauth.authUrl = "";
  oauth.sessionId = "";
  oauth.state = "";
  oauth.stateInput = "";
  oauth.code = "";
}

function resetForm() {
  const first = providers[0];
  if (!first) return;
  form.provider = first.name;
  form.name = "";
  form.priority = 100;
  form.concurrencyLimit = 4;
  form.modelScopeText = "";
  form.apiKey = "";
  form.authMode = first.default_auth_mode || first.auth_modes[0]?.value || "oauth";
  form.baseUrl = first.default_base_url || "";
  applyFieldDefaults(first);
  resetOAuthState();
}

function parseModelScope() {
  return form.modelScopeText
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function isFieldVisible(field: ProviderCapabilityField) {
  if (!field.visible_when) return true;
  return Object.entries(field.visible_when).every(([key, values]) => values.includes(fieldValues[key] || ""));
}

function fieldComponent(field: ProviderCapabilityField) {
  return field.type === "select" ? ElSelect : ElInput;
}

function fieldProps(field: ProviderCapabilityField) {
  if (field.type === "select") {
    return {};
  }
  if (field.type === "textarea") {
    return { type: "textarea", rows: 4, placeholder: field.placeholder || "" };
  }
  return { placeholder: field.placeholder || "" };
}

function buildFieldsPayload(storage: "credentials" | "meta", fields: ProviderCapabilityField[]) {
  const payload: Record<string, string> = {};
  for (const field of fields) {
    if ((field.storage || "credentials") !== storage) continue;
    if (!isFieldVisible(field)) continue;
    const value = (fieldValues[field.key] || "").trim();
    if (value) {
      payload[field.key] = value;
    }
  }
  return payload;
}

function validateCommon() {
  if (!form.name.trim()) {
    ElMessage.error(t("adminAccounts.accountNameRequired"));
    return false;
  }
  if (!form.baseUrl.trim()) {
    ElMessage.error(t("adminAccounts.baseUrlRequired"));
    return false;
  }
  return true;
}

function validateRequiredFields(fields: ProviderCapabilityField[]) {
  for (const field of fields) {
    if (!field.required || !isFieldVisible(field)) continue;
    if (!(fieldValues[field.key] || "").trim()) {
      ElMessage.error(`${field.label} ${t("common.requiredSuffix")}`);
      return false;
    }
  }
  return true;
}

async function startOAuth() {
  if (!validateCommon()) return;
  if (!validateRequiredFields(oauthFields.value)) return;
  state.oauthStarting = true;
  try {
    const result = await adminAPI.oauthStart({
      provider: form.provider,
      redirect_uri: "",
      meta: buildFieldsPayload("meta", oauthFields.value),
      oauth_type: fieldValues.oauth_type || "",
      project_id: fieldValues.project_id || "",
      tier_id: fieldValues.tier_id || "",
    });
    oauth.authUrl = result.auth_url;
    oauth.sessionId = result.session_id;
    oauth.state = result.state;
    oauth.stateInput = result.state;
    ElMessage.success(t("adminAccounts.oauthReady"));
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t("adminAccounts.oauthStartFailed"));
  } finally {
    state.oauthStarting = false;
  }
}

async function completeOAuthCreate() {
  if (!validateCommon()) return;
  if (!validateRequiredFields(oauthFields.value) || !validateRequiredFields(accountFields.value)) return;
  if (!oauth.sessionId || !oauth.code.trim() || !oauth.stateInput.trim()) {
    ElMessage.error(t("adminAccounts.oauthFieldsRequired"));
    return;
  }

  state.submitting = true;
  try {
    const credentials = await adminAPI.oauthExchange({
      session_id: oauth.sessionId,
      state: oauth.stateInput.trim(),
      code: oauth.code.trim(),
    });

    await adminAPI.oauthCreate({
      provider: form.provider,
      name: form.name.trim(),
      model_scope: parseModelScope(),
      base_url: form.baseUrl.trim(),
      priority: Number(form.priority ?? 100),
      concurrency_limit: Number(form.concurrencyLimit ?? 4),
      credentials: {
        ...(credentials as AccountCredentials),
        ...buildFieldsPayload("credentials", accountFields.value),
      },
    });

    ElMessage.success(t("adminAccounts.createSuccess"));
    emit("created");
    handleClose();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t("adminAccounts.createFailed"));
  } finally {
    state.submitting = false;
  }
}

async function createAPIKeyAccount() {
  if (!validateCommon()) return;
  if (!form.apiKey.trim()) {
    ElMessage.error(t("adminAccounts.apiKeyRequired"));
    return;
  }
  if (!validateRequiredFields(accountFields.value)) return;

  state.submitting = true;
  try {
    await adminAPI.createAccount({
      provider: form.provider,
      name: form.name.trim(),
      auth_type: form.authMode,
      base_url: form.baseUrl.trim(),
      model_scope: parseModelScope(),
      credentials: {
        api_key: form.apiKey.trim(),
        ...buildFieldsPayload("credentials", accountFields.value),
      },
      priority: Number(form.priority ?? 100),
      concurrency_limit: Number(form.concurrencyLimit ?? 4),
      metadata: {},
    });
    ElMessage.success(t("adminAccounts.createSuccess"));
    emit("created");
    handleClose();
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t("adminAccounts.createFailed"));
  } finally {
    state.submitting = false;
  }
}

async function copyText(text: string) {
  await navigator.clipboard.writeText(text);
  ElMessage.success(t("adminAccounts.copied"));
}

function handleClose() {
  emit("close");
}
</script>

<style scoped>
.wizard-shell {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.wizard-intro {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: linear-gradient(135deg, rgba(255, 255, 255, 0.95), rgba(244, 246, 252, 0.92));
}

.wizard-title {
  margin: 0 0 4px;
  font-size: 17px;
  font-weight: 700;
  color: var(--text-primary);
}

.wizard-subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}

.wizard-provider-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-full);
  background: var(--bg-raised);
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  font-weight: 600;
  white-space: nowrap;
}

.provider-notice {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 14px;
  border-radius: var(--radius-lg);
  background: var(--primary-lighter);
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.provider-notice strong {
  color: var(--text-primary);
}

.wizard-provider-chip__dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: var(--color-primary);
}

.form-grid {
  display: grid;
  gap: 16px;
}

.form-grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-grid--two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.modern-form :deep(.el-form-item) {
  margin-bottom: 18px;
}

.full-width {
  width: 100%;
}

.field-hint {
  margin-top: 6px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.4;
}

.oauth-panel,
.apikey-panel {
  padding: 16px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--bg-raised);
}

.oauth-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.oauth-panel__header h4 {
  margin: 0 0 4px;
}

.oauth-panel__header p {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.oauth-result {
  margin-bottom: 16px;
}

.oauth-state-row {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  color: var(--text-secondary);
  font-size: 13px;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 900px) {
  .form-grid--three,
  .form-grid--two {
    grid-template-columns: 1fr;
  }

  .wizard-intro,
  .oauth-panel__header {
    flex-direction: column;
  }
}
</style>
