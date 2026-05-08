<template>
  <div class="page-container">
    <div class="content-grid">
      <div class="surface-card create-section">
        <div class="card-header">
          <h3 class="card-title">
            <KeyRound :size="20" />
            {{ t('keys.createNewApiKey') }}
          </h3>
        </div>
        <div class="card-body">
          <el-form label-position="top" class="key-form">
            <div class="form-row">
              <el-form-item :label="t('keys.keyName')" class="form-item">
                <el-input v-model="form.name" :placeholder="t('keys.keyNamePlaceholder')" size="large" />
              </el-form-item>
              <el-form-item :label="t('keys.provider')" class="form-item">
                <ElSelect v-model="form.provider" filterable clearable size="large" style="width: 100%" :placeholder="t('keys.selectProvider')">
                  <ElOption v-for="provider in providerOptions" :key="provider.value" :label="provider.label" :value="provider.value" />
                </ElSelect>
              </el-form-item>
            </div>
            <div class="form-actions">
              <el-button type="primary" size="large" @click="create">
                <KeyRound :size="18" />
                {{ t('keys.createKey') }}
              </el-button>
            </div>
          </el-form>
        </div>
      </div>

      <transition name="slide-fade">
        <div v-if="lastCreatedKey" class="surface-card success-alert">
          <div class="alert-header">
            <div class="alert-icon">
              <ShieldCheck :size="20" />
            </div>
            <div class="alert-content">
              <h4>{{ t('keys.keyCreatedSuccess') }}</h4>
              <p>{{ t('keys.keyCreatedWarning') }}</p>
            </div>
            <el-button text @click="lastCreatedKey = null">
              <X :size="18" />
            </el-button>
          </div>
          <div class="secret-display">
            <code class="secret-key mono">{{ lastCreatedKey.secret }}</code>
          </div>
          <div class="secret-actions">
            <el-button type="primary" @click="copySecret(lastCreatedKey.secret)">
              <Copy :size="16" />
              {{ t('keys.copySecret') }}
            </el-button>
            <el-button @click="lastCreatedKey = null">{{ t('common.close') }}</el-button>
          </div>
        </div>
      </transition>

      <div class="surface-card keys-section">
        <div class="card-header">
          <h3 class="card-title">
            <ShieldCheck :size="20" />
            {{ t('keys.myApiKeys') }}
          </h3>
          <span class="key-count">{{ keys.length }} {{ t('keys.keyCount') }}</span>
        </div>
        <div class="card-body">
          <el-table :data="keys" :empty-text="t('common.noKeys')" class="modern-table" :stripe="true">
            <el-table-column prop="name" :label="t('keys.name')" width="100">
              <template #default="{ row }">
                <div class="key-name">
                  <KeyRound :size="16" />
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="t('keys.secret')" min-width="200">
              <template #default="{ row }">
                <div class="secret-row">
                  <code class="secret-value mono">
                    {{ revealed[row.id] ? row.secret : maskSecret(row.secret) }}
                  </code>
                  <div class="secret-actions-inline">
                    <el-button text size="small" @click="revealed[row.id] = !revealed[row.id]">
                      <component :is="revealed[row.id] ? EyeOff : Eye" :size="14" />
                      {{ revealed[row.id] ? t('keys.hide') : t('keys.show') }}
                    </el-button>
                    <el-button text size="small" type="primary" @click="copySecret(row.secret)">
                      <Copy :size="14" />
                      {{ t('common.copy') }}
                    </el-button>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="t('keys.status')" width="80">
              <template #default="{ row }">
                <el-tag :type="isActiveStatus(row.status) ? 'success' : 'info'">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="provider" :label="t('keys.provider')" width="120">
              <template #default="{ row }">
                <el-tag size="small" type="info" effect="plain">{{ row.provider }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('keys.lastUsed')" width="130">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.last_used_at_ms) }}</span>
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
import { Copy, Eye, EyeOff, KeyRound, ShieldCheck, X } from "lucide-vue-next";
import { ElButton, ElForm, ElFormItem, ElInput, ElOption, ElSelect, ElTable, ElTableColumn, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import type { APIKey, ModelCatalogChannel } from "@/api/types";
import { formatTime, isActiveStatus, maskSecret } from "@/utils";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const keys = ref<APIKey[]>([]);
const catalog = ref<ModelCatalogChannel[]>([]);
const providers = ref<string[]>([]);
const lastCreatedKey = ref<APIKey | null>(null);
const revealed = reactive<Record<number, boolean>>({});
const form = reactive({
  name: "",
  provider: "",
});

const providerOptions = computed(() =>
  providers.value.map((key) => {
    const matched = catalog.value.find((item) => item.key === key);
    return {
      value: key,
      label: matched?.name || key,
    };
  }),
);

async function load() {
  keys.value = await userAPI.keys();
}

async function loadCatalog() {
  try {
    catalog.value = await userAPI.modelCatalog();
  } catch {
    catalog.value = [];
  }
}

async function loadProviders() {
  try {
    providers.value = await userAPI.providers();
  } catch {
    providers.value = [];
  }
}

async function create() {
  if (!form.provider) return;
  const item = await userAPI.createKey({
    provider: form.provider,
    name: form.name,
  });
  lastCreatedKey.value = item;
  revealed[item.id] = true;
  form.name = "";
  form.provider = "";
  await load();
}

function copySecret(value: string) {
  void navigator.clipboard.writeText(value);
}

onMounted(() => {
  void load();
  void loadCatalog();
  void loadProviders();
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

.create-section {
  background: var(--bg-raised);
}

.key-form {
  padding: 0;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 24px;
}

.form-item {
  margin-bottom: 0;
}

.form-actions {
  display: flex;
  justify-content: flex-start;
}

.form-actions .el-button {
  min-width: 160px;
}

.success-alert {
  border: 2px solid var(--success-color);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.03), rgba(16, 185, 129, 0.06));
}

.alert-header {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
}

.alert-icon {
  width: 44px;
  height: 44px;
  background: var(--success-light);
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--success-color);
  flex-shrink: 0;
}

.alert-content {
  flex: 1;
}

.alert-content h4 {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.alert-content p {
  margin: 0;
  font-size: 13px;
  color: var(--text-muted);
}

.secret-display {
  background: var(--border-light);
  border-radius: var(--radius-lg);
  padding: 16px 20px;
  margin-bottom: 20px;
}

.secret-key {
  font-size: 14px;
  color: var(--text-primary);
  word-break: break-all;
}

.secret-actions {
  display: flex;
  gap: 12px;
}

.keys-section {
  overflow: visible;
}

.key-count {
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

.key-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
  color: var(--text-primary);
}

.key-name svg {
  color: var(--primary-color);
}

.secret-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.secret-value {
  font-size: 13px;
  color: var(--text-secondary);
  background: var(--border-light);
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.secret-actions-inline {
  display: flex;
  gap: 8px;
}

.models-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.models-text,
.time-text {
  font-size: 13px;
  color: var(--text-muted);
}

.slide-fade-enter-active {
  transition: all 0.3s ease-out;
}

.slide-fade-leave-active {
  transition: all 0.2s ease-in;
}

.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateY(-10px);
  opacity: 0;
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
