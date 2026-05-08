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
          <span>{{ currentProvider.label }}</span>
        </div>
      </div>

      <div class="provider-notice">
        <strong>{{ currentProvider.label }}</strong>
        <span>{{ providerNotice }}</span>
      </div>

      <el-form label-position="top" class="modern-form">
        <div class="form-grid form-grid--three">
          <el-form-item :label="t('adminAccounts.provider')">
            <el-select v-model="form.provider">
              <el-option v-for="item in providers" :key="item.value" :label="item.label" :value="item.value" />
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
            <el-input v-model="form.baseUrl" :placeholder="currentProvider.baseUrlPlaceholder" />
          </el-form-item>

          <el-form-item :label="t('adminAccounts.priority')">
            <el-input-number v-model="form.priority" :min="1" :max="10000" class="full-width" />
          </el-form-item>

          <el-form-item :label="t('adminAccounts.concurrencyLimit')">
            <el-input-number v-model="form.concurrencyLimit" :min="0" :max="128" class="full-width" />
            <div class="field-hint">{{ form.concurrencyLimit === 0 ? t('adminAccounts.unlimitedHint') : t('adminAccounts.concurrencyLimitCreateHint') }}</div>
          </el-form-item>
        </div>

        <div v-if="showGeminiTier" class="form-grid form-grid--three">
          <el-form-item :label="t('adminAccounts.geminiTier')">
            <el-select v-model="form.tierId">
              <el-option v-for="item in geminiTierOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
          </el-form-item>
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

          <div v-if="form.provider === 'gemini'" class="form-grid form-grid--two">
            <el-form-item :label="t('adminAccounts.geminiOAuthType')">
              <el-select v-model="form.oauthType">
                <el-option v-for="item in geminiOAuthTypes" :key="item.value" :label="item.label" :value="item.value" />
              </el-select>
            </el-form-item>

            <el-form-item :label="t('adminAccounts.geminiProjectId')">
              <el-input v-model="form.projectId" :placeholder="t('adminAccounts.geminiProjectIdPlaceholder')" />
            </el-form-item>
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
          <el-form-item :label="apiCredentialLabel">
            <el-input
              v-model="form.apiKey"
              type="textarea"
              :rows="4"
              :placeholder="apiCredentialPlaceholder"
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
import type { AccountCredentials } from "@/api/types";

type ProviderKey = "openai" | "claude" | "gemini" | "antigravity";
type AuthMode = "api_key" | "oauth";

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  close: [];
  created: [];
}>();

const { t } = useI18n();

const providers = [
  { value: "openai", label: "OpenAI", baseUrl: "https://api.openai.com", baseUrlPlaceholder: "https://api.openai.com" },
  { value: "claude", label: "Claude", baseUrl: "https://api.anthropic.com", baseUrlPlaceholder: "https://api.anthropic.com" },
  { value: "gemini", label: "Gemini", baseUrl: "https://generativelanguage.googleapis.com", baseUrlPlaceholder: "https://generativelanguage.googleapis.com" },
  { value: "antigravity", label: "Antigravity", baseUrl: "", baseUrlPlaceholder: "https://your-upstream.example.com" },
] as const;

const geminiOAuthTypes = [
  { value: "code_assist", label: "Code Assist" },
  { value: "google_one", label: "Google One" },
  { value: "ai_studio", label: "AI Studio" },
];

const geminiTierMap: Record<string, Array<{ value: string; label: string }>> = {
  code_assist: [
    { value: "gcp_standard", label: "GCP Standard" },
    { value: "gcp_enterprise", label: "GCP Enterprise" },
  ],
  google_one: [
    { value: "google_one_free", label: "Google One Free" },
    { value: "google_ai_pro", label: "Google AI Pro" },
    { value: "google_ai_ultra", label: "Google AI Ultra" },
  ],
  ai_studio: [
    { value: "aistudio_free", label: "AI Studio Free" },
    { value: "aistudio_paid", label: "AI Studio Paid" },
  ],
};

const form = reactive({
  provider: "openai" as ProviderKey,
  authMode: "oauth" as AuthMode,
  name: "",
  baseUrl: "https://api.openai.com",
  priority: 100,
  concurrencyLimit: 4,
  modelScopeText: "",
  apiKey: "",
  oauthType: "code_assist",
  projectId: "",
  tierId: "gcp_standard",
});

const oauth = reactive({
  authUrl: "",
  sessionId: "",
  state: "",
  stateInput: "",
  code: "",
});

const oauthStarting = computed(() => state.oauthStarting);
const submitting = computed(() => state.submitting);

const state = reactive({
  oauthStarting: false,
  submitting: false,
});

const currentProvider = computed(() => providers.find((item) => item.value === form.provider) || providers[0]);
const isOAuthMode = computed(() => form.authMode === "oauth");
const showGeminiTier = computed(() => form.provider === "gemini");
const authModeOptions = computed(() => [
  { value: "oauth", label: t("adminAccounts.oauthMode") },
  { value: "api_key", label: form.provider === "antigravity" ? t("adminAccounts.upstreamMode") : t("adminAccounts.apiKeyMode") },
]);
const providerNotice = computed(() => {
  if (form.provider === "openai") return t("adminAccounts.openaiNotice");
  if (form.provider === "claude") return t("adminAccounts.claudeNotice");
  if (form.provider === "gemini") return t("adminAccounts.geminiNotice");
  return t("adminAccounts.antigravityNotice");
});
const geminiTierOptions = computed(() => {
  if (isOAuthMode.value) {
    return geminiTierMap[form.oauthType] || geminiTierMap.code_assist;
  }
  return geminiTierMap.ai_studio;
});

const apiCredentialLabel = computed(() =>
  form.provider === "antigravity" ? t("adminAccounts.upstreamApiKey") : t("adminAccounts.apiKey"),
);

const apiCredentialPlaceholder = computed(() =>
  form.provider === "antigravity" ? t("adminAccounts.upstreamApiKeyPlaceholder") : t("adminAccounts.apiKeyPlaceholder"),
);

watch(
  () => form.provider,
  (provider) => {
    form.baseUrl = providers.find((item) => item.value === provider)?.baseUrl || "";
    form.authMode = "oauth";
    if (provider !== "gemini") {
      form.oauthType = "code_assist";
      form.projectId = "";
      form.tierId = "gcp_standard";
    } else {
      form.tierId = "gcp_standard";
    }
    resetOAuthState();
  },
);

watch(
  () => form.authMode,
  (mode) => {
    if (form.provider === "antigravity" && mode === "api_key" && !form.baseUrl.trim()) {
      form.baseUrl = "";
    }
    if (form.provider === "gemini" && mode === "api_key") {
      form.tierId = "aistudio_free";
    } else if (form.provider === "gemini" && mode === "oauth") {
      form.tierId = form.oauthType === "google_one" ? "google_one_free" : form.oauthType === "ai_studio" ? "aistudio_free" : "gcp_standard";
    }
  },
);

watch(
  () => form.oauthType,
  (oauthType) => {
    if (form.provider !== "gemini" || !isOAuthMode.value) return;
    form.tierId = oauthType === "google_one" ? "google_one_free" : oauthType === "ai_studio" ? "aistudio_free" : "gcp_standard";
  },
);

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      resetForm();
    }
  },
);

function resetOAuthState() {
  oauth.authUrl = "";
  oauth.sessionId = "";
  oauth.state = "";
  oauth.stateInput = "";
  oauth.code = "";
}

function resetForm() {
  form.provider = "openai";
  form.authMode = "oauth";
  form.name = "";
  form.baseUrl = "https://api.openai.com";
  form.priority = 100;
  form.concurrencyLimit = 4;
  form.modelScopeText = "";
  form.apiKey = "";
  form.oauthType = "code_assist";
  form.projectId = "";
  form.tierId = "gcp_standard";
  resetOAuthState();
}

function parseModelScope() {
  return form.modelScopeText
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function buildCreatePayload(credentials: Record<string, unknown>) {
  return {
    provider: form.provider,
    name: form.name.trim(),
    auth_type: form.authMode,
    base_url: form.baseUrl.trim(),
    model_scope: parseModelScope(),
    credentials,
    priority: Number(form.priority ?? 100),
    concurrency_limit: Number(form.concurrencyLimit ?? 4),
    metadata: {},
  };
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

async function startOAuth() {
  if (!validateCommon()) return;
  state.oauthStarting = true;
  try {
    const result = await adminAPI.oauthStart({
      provider: form.provider,
      redirect_uri: "",
      oauth_type: form.provider === "gemini" ? form.oauthType : "",
      project_id: form.provider === "gemini" ? form.projectId.trim() : "",
      tier_id: form.provider === "gemini" ? form.tierId : "",
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
      credentials: credentials as AccountCredentials,
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

  state.submitting = true;
  try {
    await adminAPI.createAccount(
      buildCreatePayload({
        api_key: form.apiKey.trim(),
        base_url: form.baseUrl.trim(),
        tier_id: form.provider === "gemini" ? form.tierId : undefined,
      }),
    );
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
  width: 9px;
  height: 9px;
  border-radius: 999px;
  background: var(--primary-color);
}

.wizard-provider-chip__dot.openai {
  background: #10a37f;
}

.wizard-provider-chip__dot.claude {
  background: #d4a574;
}

.wizard-provider-chip__dot.gemini {
  background: #4285f4;
}

.wizard-provider-chip__dot.antigravity {
  background: #7c3aed;
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

.oauth-panel,
.apikey-panel {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-xl);
  background: var(--bg-soft);
  padding: 18px;
}

.oauth-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}

.oauth-panel__header h4 {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
}

.oauth-panel__header p {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

.oauth-result {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 18px;
}

.oauth-state-row {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  font-size: 12px;
  color: var(--text-secondary);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.full-width {
  width: 100%;
}

.field-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--text-muted);
  line-height: 1.5;
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
