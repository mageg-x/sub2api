<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">账户总数</p>
        <p class="stat-value">{{ accounts.length }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">当前上游账户池</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">OAuth 账户</p>
        <p class="stat-value">{{ oauthActiveCount }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">可自动刷新 token</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">活跃账户</p>
        <p class="stat-value">{{ activeAccounts }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">参与请求转发</p>
      </ElCard>
    </div>

    <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon />

    <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <KeyRound :size="16" />
            <span>静态账户</span>
          </div>
        </template>
        <ElForm label-position="top">
          <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
            <ElFormItem label="Provider">
              <ElInput v-model="staticForm.provider" placeholder="openai / claude / gemini / antigravity" />
            </ElFormItem>
            <ElFormItem label="认证方式">
              <ElInput v-model="staticForm.auth_type" placeholder="api_key / static" />
            </ElFormItem>
          </div>
          <ElFormItem label="账户名">
            <ElInput v-model="staticForm.name" placeholder="如 openai-main-01" />
          </ElFormItem>
          <ElFormItem label="Base URL">
            <ElInput v-model="staticForm.base_url" placeholder="可留空" />
          </ElFormItem>
          <ElFormItem label="模型范围">
            <ElInput v-model="staticForm.model_scope" placeholder="逗号分隔模型白名单" />
          </ElFormItem>
          <ElFormItem label="密钥 / Token">
            <ElInput v-model="staticForm.api_key" type="textarea" :rows="4" placeholder="api_key 或 access_token" />
          </ElFormItem>
          <ElButton type="primary" :loading="loading" @click="createStaticAccount">创建静态账户</ElButton>
        </ElForm>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <Link2 :size="16" />
            <span>OAuth 账户</span>
          </div>
        </template>
        <ElForm label-position="top">
          <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
            <ElFormItem label="Provider">
              <ElInput v-model="oauthForm.provider" placeholder="openai / claude / gemini / antigravity" />
            </ElFormItem>
            <ElFormItem label="oauth_type">
              <ElInput v-model="oauthForm.oauth_type" placeholder="如 ai_studio / code_assist，可空" />
            </ElFormItem>
          </div>
          <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
            <ElFormItem label="redirect_uri">
              <ElInput v-model="oauthForm.redirect_uri" placeholder="可留空" />
            </ElFormItem>
            <ElFormItem label="project_id / tier_id">
              <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px">
                <ElInput v-model="oauthForm.project_id" placeholder="project_id" />
                <ElInput v-model="oauthForm.tier_id" placeholder="tier_id" />
              </div>
            </ElFormItem>
          </div>
          <ElButton type="primary" plain @click="startOAuth">生成授权链接</ElButton>
          <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; margin-top: 16px">
            <ElFormItem label="session_id">
              <ElInput v-model="oauthForm.session_id" />
            </ElFormItem>
            <ElFormItem label="state">
              <ElInput v-model="oauthForm.state" />
            </ElFormItem>
          </div>
          <ElFormItem label="code">
            <ElInput v-model="oauthForm.code" type="textarea" :rows="3" placeholder="回调后的 code 粘贴到这里" />
          </ElFormItem>
          <ElButton type="success" plain @click="exchangeOAuth">交换 token</ElButton>
          <div class="pre-box" style="margin-top: 16px">
            {{ prettyJSON(oauthCredentials) }}
          </div>

          <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; margin-top: 16px">
            <ElFormItem label="账户名">
              <ElInput v-model="oauthForm.name" placeholder="如 claude-oauth-01" />
            </ElFormItem>
            <ElFormItem label="Base URL">
              <ElInput v-model="oauthForm.base_url" placeholder="可留空" />
            </ElFormItem>
            <ElFormItem label="模型范围">
              <ElInput v-model="oauthForm.model_scope" placeholder="逗号分隔模型白名单" />
            </ElFormItem>
            <ElFormItem label="priority / concurrency">
              <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px">
                <ElInput v-model.number="oauthForm.priority" type="number" />
                <ElInput v-model.number="oauthForm.concurrency_limit" type="number" />
              </div>
            </ElFormItem>
          </div>
          <ElButton type="primary" :loading="loading" @click="createOAuthAccount">创建 OAuth 账户</ElButton>
        </ElForm>
      </ElCard>
    </div>

    <div style="display: grid; grid-template-columns: 1.2fr 0.8fr; gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <Boxes :size="16" />
            <span>账户列表</span>
          </div>
        </template>
        <ElTable :data="accounts" empty-text="暂无账户">
          <ElTableColumn prop="id" label="ID" width="80" />
          <ElTableColumn prop="provider" label="Provider" width="120" />
          <ElTableColumn prop="name" label="名称" min-width="180" />
          <ElTableColumn prop="auth_type" label="认证方式" width="120" />
          <ElTableColumn label="状态" width="110">
            <template #default="{ row }">
              <ElTag :type="isActiveStatus(row.status) ? 'success' : 'info'">{{ row.status }}</ElTag>
            </template>
          </ElTableColumn>
          <ElTableColumn prop="priority" label="优先级" width="100" />
          <ElTableColumn prop="concurrency_limit" label="并发" width="90" />
          <ElTableColumn label="Token 到期" min-width="180">
            <template #default="{ row }">
              {{ formatTime(row.credentials?.expires_at_ms || row.expires_at_ms) }}
            </template>
          </ElTableColumn>
          <ElTableColumn label="最近刷新" min-width="180">
            <template #default="{ row }">
              {{ formatTime(row.last_refreshed_at_ms) }}
            </template>
          </ElTableColumn>
          <ElTableColumn prop="base_url" label="Base URL" min-width="220" show-overflow-tooltip />
          <ElTableColumn prop="model_scope_json" label="模型范围" min-width="220" show-overflow-tooltip />
          <ElTableColumn label="操作" min-width="250">
            <template #default="{ row }">
              <div style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap">
                <ElButton text type="primary" @click="selectAccount(row)">编辑</ElButton>
                <ElButton text @click="toggleAccountStatus(row, isActiveStatus(row.status) ? 'disabled' : 'active')">
                  {{ isActiveStatus(row.status) ? "停用" : "启用" }}
                </ElButton>
                <ElButton v-if="row.auth_type === 'oauth'" text type="success" :loading="refreshingID === row.id" @click="refreshAccount(row)"> 刷新 Token </ElButton>
              </div>
            </template>
          </ElTableColumn>
        </ElTable>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <span>编辑账户</span>
        </template>
        <ElForm label-position="top">
          <ElFormItem label="账户 ID">
            <ElInput :model-value="selectedAccountID || ''" readonly />
          </ElFormItem>
          <ElFormItem label="状态">
            <ElInput v-model="updateForm.status" placeholder="active / disabled / error" />
          </ElFormItem>
          <ElFormItem label="优先级">
            <ElInput v-model.number="updateForm.priority" type="number" />
          </ElFormItem>
          <ElFormItem label="并发限制">
            <ElInput v-model.number="updateForm.concurrency_limit" type="number" />
          </ElFormItem>
          <ElButton type="primary" :disabled="!selectedAccountID" :loading="updateSubmitting" @click="saveAccount">保存账户设置</ElButton>
        </ElForm>

        <div v-if="selectedAccount" style="display: grid; gap: 12px; margin-top: 18px">
          <ElDescriptions :column="1" border>
            <ElDescriptionsItem label="Provider">{{ selectedAccount.provider }}</ElDescriptionsItem>
            <ElDescriptionsItem label="账户名">{{ selectedAccount.name }}</ElDescriptionsItem>
            <ElDescriptionsItem label="认证方式">{{ selectedAccount.auth_type }}</ElDescriptionsItem>
            <ElDescriptionsItem label="Token 到期">{{ formatTime(selectedAccount.credentials?.expires_at_ms || selectedAccount.expires_at_ms) }}</ElDescriptionsItem>
            <ElDescriptionsItem label="最近刷新">{{ formatTime(selectedAccount.last_refreshed_at_ms) }}</ElDescriptionsItem>
            <ElDescriptionsItem label="邮箱">{{ selectedAccount.credentials?.email || "-" }}</ElDescriptionsItem>
            <ElDescriptionsItem label="OAuth 类型">{{ selectedAccount.credentials?.oauth_type || "-" }}</ElDescriptionsItem>
            <ElDescriptionsItem label="项目 / 组织">{{ selectedAccount.credentials?.project_id || selectedAccount.credentials?.organization_id || "-" }}</ElDescriptionsItem>
            <ElDescriptionsItem label="订阅 / 计划">{{ selectedAccount.credentials?.plan_type || selectedAccount.credentials?.subscription_until || "-" }}</ElDescriptionsItem>
          </ElDescriptions>

          <div class="pre-box">{{ prettyJSON(selectedAccount.credentials) }}</div>
        </div>
      </ElCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { Boxes, KeyRound, Link2 } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElDescriptions, ElDescriptionsItem, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { Account, AccountCredentials } from "@/api/types";
import { formatTime, isActiveStatus, parseCSV, prettyJSON } from "@/utils";

const accounts = ref<Account[]>([]);
const error = ref("");
const loading = ref(false);
const oauthCredentials = ref<AccountCredentials | null>(null);
const updateSubmitting = ref(false);
const refreshingID = ref(0);

const staticForm = reactive({
  provider: "openai",
  auth_type: "api_key",
  name: "",
  base_url: "",
  model_scope: "",
  api_key: "",
});

const oauthForm = reactive({
  provider: "openai",
  oauth_type: "",
  redirect_uri: "",
  project_id: "",
  tier_id: "",
  session_id: "",
  state: "",
  code: "",
  name: "",
  base_url: "",
  model_scope: "",
  priority: 100,
  concurrency_limit: 4,
});

const oauthActiveCount = computed(() => accounts.value.filter((item) => item.auth_type === "oauth").length);
const activeAccounts = computed(() => accounts.value.filter((item) => isActiveStatus(item.status)).length);
const selectedAccountID = ref(0);
const selectedAccount = computed(() => accounts.value.find((item) => item.id === selectedAccountID.value) || null);
const updateForm = reactive({
  status: "active",
  priority: 100,
  concurrency_limit: 4,
});

async function load() {
  accounts.value = await adminAPI.accounts();
}

async function createStaticAccount() {
  error.value = "";
  loading.value = true;
  try {
    await adminAPI.createAccount({
      provider: staticForm.provider,
      name: staticForm.name,
      auth_type: staticForm.auth_type,
      base_url: staticForm.base_url,
      model_scope: parseCSV(staticForm.model_scope),
      credentials: { api_key: staticForm.api_key },
    });
    staticForm.name = "";
    staticForm.base_url = "";
    staticForm.model_scope = "";
    staticForm.api_key = "";
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "create failed";
  } finally {
    loading.value = false;
  }
}

async function startOAuth() {
  error.value = "";
  try {
    const result = await adminAPI.oauthStart({
      provider: oauthForm.provider,
      oauth_type: oauthForm.oauth_type,
      redirect_uri: oauthForm.redirect_uri,
      project_id: oauthForm.project_id,
      tier_id: oauthForm.tier_id,
    });
    oauthForm.session_id = result.session_id;
    oauthForm.state = result.state;
    window.open(result.auth_url, "_blank", "noopener");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "oauth start failed";
  }
}

async function exchangeOAuth() {
  error.value = "";
  try {
    oauthCredentials.value = await adminAPI.oauthExchange({
      session_id: oauthForm.session_id,
      state: oauthForm.state,
      code: oauthForm.code,
    });
  } catch (err) {
    error.value = err instanceof Error ? err.message : "oauth exchange failed";
  }
}

async function createOAuthAccount() {
  if (!oauthCredentials.value) {
    error.value = "请先完成 OAuth 交换";
    return;
  }
  error.value = "";
  loading.value = true;
  try {
    await adminAPI.oauthCreate({
      provider: oauthForm.provider,
      name: oauthForm.name,
      base_url: oauthForm.base_url,
      model_scope: parseCSV(oauthForm.model_scope),
      priority: Number(oauthForm.priority || 100),
      concurrency_limit: Number(oauthForm.concurrency_limit || 4),
      credentials: oauthCredentials.value,
    });
    oauthForm.name = "";
    oauthForm.base_url = "";
    oauthForm.model_scope = "";
    oauthForm.code = "";
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "oauth account create failed";
  } finally {
    loading.value = false;
  }
}

function selectAccount(account: Account) {
  selectedAccountID.value = account.id;
  updateForm.status = account.status;
  updateForm.priority = account.priority;
  updateForm.concurrency_limit = account.concurrency_limit;
}

async function toggleAccountStatus(account: Account, status: string) {
  error.value = "";
  try {
    await adminAPI.updateAccount(account.id, { status });
    await load();
    if (selectedAccountID.value === account.id) {
      const next = accounts.value.find((item) => item.id === account.id);
      if (next) selectAccount(next);
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "account status update failed";
  }
}

async function refreshAccount(account: Account) {
  error.value = "";
  refreshingID.value = account.id;
  try {
    const refreshed = await adminAPI.refreshAccount(account.id);
    const index = accounts.value.findIndex((item) => item.id === account.id);
    if (index >= 0) {
      accounts.value[index] = refreshed;
    }
    selectAccount(refreshed);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "oauth refresh failed";
  } finally {
    refreshingID.value = 0;
  }
}

async function saveAccount() {
  if (!selectedAccountID.value) return;
  error.value = "";
  updateSubmitting.value = true;
  try {
    await adminAPI.updateAccount(selectedAccountID.value, {
      status: updateForm.status,
      priority: Number(updateForm.priority || 0),
      concurrency_limit: Number(updateForm.concurrency_limit || 0),
    });
    await load();
    const next = accounts.value.find((item) => item.id === selectedAccountID.value);
    if (next) selectAccount(next);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "account update failed";
  } finally {
    updateSubmitting.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
