<template>
  <el-dialog
    :model-value="visible"
    :title="t('adminAccounts.editAccount')"
    width="760px"
    destroy-on-close
    @close="$emit('close')"
  >
    <el-form label-position="top" class="modern-form">
      <div class="form-grid form-grid--two">
        <el-form-item :label="t('adminAccounts.baseUrl')">
          <el-input v-model="form.baseUrl" :placeholder="t('adminAccounts.baseUrlRequired')" />
        </el-form-item>

        <el-form-item :label="t('adminAccounts.status')">
          <el-select v-model="form.status">
            <el-option :label="t('common.active')" value="active" />
            <el-option :label="t('common.disabled')" value="disabled" />
          </el-select>
        </el-form-item>

        <el-form-item :label="t('adminAccounts.priority')">
          <el-input-number v-model="form.priority" :min="1" :max="10000" class="full-width" />
        </el-form-item>

        <el-form-item :label="t('adminAccounts.concurrencyLimit')">
          <el-input-number v-model="form.concurrencyLimit" :min="0" :max="128" class="full-width" />
          <div class="field-hint">{{ form.concurrencyLimit === 0 ? t('adminAccounts.unlimitedHint') : t('adminAccounts.concurrencyLimitCreateHint') }}</div>
        </el-form-item>
      </div>

      <el-form-item :label="t('adminAccounts.modelScope')">
        <el-input
          v-model="form.modelScopeText"
          type="textarea"
          :rows="4"
          :placeholder="t('adminAccounts.modelScopePlaceholder')"
        />
        <div class="field-hint">{{ t('adminAccounts.modelScopeEditHint') }}</div>
      </el-form-item>

      <div class="readonly-note">
        {{ t('adminAccounts.credentialsReadonlyHint') }}
      </div>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="$emit('close')">{{ t('common.cancel') }}</el-button>
        <el-button :loading="submitting" type="primary" @click="save">{{ t('common.save') }}</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from "vue";
import { ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElMessage, ElOption, ElSelect } from "element-plus";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api/admin";
import type { Account } from "@/api/types";

const props = defineProps<{
  visible: boolean;
  account: Account | null;
}>();

const emit = defineEmits<{
  close: [];
  updated: [];
}>();

const { t } = useI18n();

const form = reactive({
  baseUrl: "",
  status: "active",
  priority: 100,
  concurrencyLimit: 4,
  modelScopeText: "",
});

const state = reactive({
  submitting: false,
});

function parseModelScopeText(raw: string) {
  return raw
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseModelScopeJSON(raw: string) {
  const text = String(raw || "").trim();
  if (!text) return [] as string[];
  try {
    const parsed = JSON.parse(text);
    if (Array.isArray(parsed)) {
      return parsed.map((item) => String(item).trim()).filter(Boolean);
    }
  } catch {
    return parseModelScopeText(text);
  }
  return [];
}

watch(
  () => props.account,
  (account) => {
    form.baseUrl = account?.base_url || "";
    form.status = account?.status || "active";
    form.priority = account?.priority ?? 100;
    form.concurrencyLimit = account?.concurrency_limit ?? 4;
    form.modelScopeText = parseModelScopeJSON(account?.model_scope_json || "").join("\n");
  },
  { immediate: true },
);

async function save() {
  if (!props.account) return;
  if (!form.baseUrl.trim()) {
    ElMessage.error(t('adminAccounts.baseUrlRequired'));
    return;
  }

  state.submitting = true;
  try {
    await adminAPI.updateAccount(props.account.id, {
      base_url: form.baseUrl.trim(),
      status: form.status,
      priority: Number(form.priority ?? 100),
      concurrency_limit: Number(form.concurrencyLimit ?? 4),
      model_scope_json: JSON.stringify(parseModelScopeText(form.modelScopeText)),
    });
    ElMessage.success(t('adminAccounts.updateSuccess'));
    emit('updated');
    emit('close');
  } catch (err) {
    ElMessage.error(err instanceof Error ? err.message : t('adminAccounts.updateFailed'));
  } finally {
    state.submitting = false;
  }
}

const submitting = computed(() => state.submitting);
</script>

<style scoped>
.form-grid {
  display: grid;
  gap: 16px;
}

.form-grid--two {
  grid-template-columns: repeat(2, minmax(0, 1fr));
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

.readonly-note {
  padding: 12px 14px;
  border-radius: var(--radius-lg);
  background: var(--bg-soft);
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.6;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

@media (max-width: 760px) {
  .form-grid--two {
    grid-template-columns: 1fr;
  }
}
</style>
