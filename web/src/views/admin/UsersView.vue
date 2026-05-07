<template>
  <div class="page-container">
    <div class="content-grid">
      <div class="stats-row">
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon users">
            <Users :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">总用户数</span>
            <span class="stat-mini-value">{{ users.length }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon active">
            <UserCheck :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">活跃用户</span>
            <span class="stat-mini-value">{{ activeUsers }}</span>
          </div>
        </div>
        <div class="surface-card stat-mini">
          <div class="stat-mini-icon balance">
            <CircleDollarSign :size="20" />
          </div>
          <div class="stat-mini-content">
            <span class="stat-mini-label">余额总览</span>
            <span class="stat-mini-value">{{ formatCurrency(users.reduce((sum, item) => sum + item.balance, 0)) }} 元</span>
          </div>
        </div>
      </div>

      <div v-if="error" class="surface-card error-banner">
        <el-alert :title="error" type="error" :closable="false" show-icon />
      </div>

      <div class="surface-card users-section">
        <div class="card-header">
          <h3 class="card-title">
            <ShieldUser :size="20" />
            用户列表
          </h3>
          <span class="user-count">{{ users.length }} 个用户</span>
        </div>
        <div class="card-body">
          <el-table :data="users" empty-text="暂无用户" class="modern-table" :stripe="true">
            <el-table-column prop="id" label="ID" width="60" />
            <el-table-column prop="email" label="邮箱" min-width="160">
              <template #default="{ row }">
                <div class="email-cell">
                  <Mail :size="14" />
                  <span>{{ row.email }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="name" label="名称" width="100">
              <template #default="{ row }">
                <div class="name-cell">
                  <User :size="14" />
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="role" label="角色" width="80">
              <template #default="{ row }">
                <el-tag :type="row.role === 'admin' ? 'danger' : 'info'" size="small">
                  {{ row.role }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80">
              <template #default="{ row }">
                <el-tag :type="isActiveStatus(row.status) ? 'success' : 'info'" size="small">
                  {{ row.status }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="余额" width="90">
              <template #default="{ row }">
                <span class="balance-value">{{ formatCurrency(row.balance) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="allowed_models_json" label="允许模型" min-width="120">
              <template #default="{ row }">
                <div v-if="parseAllowedModels(row.allowed_models_json).length" class="models-list">
                  <el-tag v-for="model in parseAllowedModels(row.allowed_models_json)" :key="model" size="small" type="info" effect="plain">
                    {{ model }}
                  </el-tag>
                </div>
                <span v-else class="models-text">全部</span>
              </template>
            </el-table-column>
            <el-table-column label="最近登录" width="140">
              <template #default="{ row }">
                <span class="time-text">{{ formatTime(row.last_login_at_ms) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ row }">
                <el-button text type="primary" @click="selectUser(row)">编辑</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
    </div>

    <ElDialog v-model="dialogVisible" title="编辑用户" width="720px" destroy-on-close>
      <ElForm label-position="top" class="create-form">
        <div class="form-row">
          <ElFormItem label="用户名" class="form-item">
            <ElInput v-model="editForm.name" size="large" />
          </ElFormItem>
          <ElFormItem label="状态" class="form-item">
            <ElSelect v-model="editForm.status" size="large">
              <ElOption label="active" value="active" />
              <ElOption label="disabled" value="disabled" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="角色" class="form-item">
            <ElSelect v-model="editForm.role" size="large">
              <ElOption label="user" value="user" />
              <ElOption label="admin" value="admin" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="余额" class="form-item">
            <ElInputNumber v-model="editForm.balance" :min="0" :step="10000" size="large" style="width: 100%" />
          </ElFormItem>
        </div>
        <div class="form-row">
          <ElFormItem label="费率百分比" class="form-item">
            <ElInputNumber v-model="editForm.rate_percent" :min="0" :max="1000" :step="1" size="large" style="width: 100%" />
          </ElFormItem>
          <ElFormItem label="重置密码" class="form-item">
            <ElInput v-model="editForm.password" type="password" show-password size="large" placeholder="留空表示不修改" />
          </ElFormItem>
        </div>
        <ElFormItem label="允许模型" class="form-item-wide">
          <ElSelect v-model="editForm.models" multiple filterable allow-create default-first-option clearable size="large" style="width: 100%" placeholder="选择或输入模型；清空表示不限制">
            <ElOption v-for="model in commonModelOptions" :key="model" :label="model" :value="model" />
          </ElSelect>
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="actions-row">
          <ElButton @click="closeDialog">取消</ElButton>
          <ElButton type="primary" :loading="saving" @click="saveUser">保存修改</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CircleDollarSign, Mail, ShieldUser, User, UserCheck, Users } from "lucide-vue-next";
import { ElAlert, ElButton, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElOption, ElSelect, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { User as UserType } from "@/api/types";
import { formatCurrency, formatTime, isActiveStatus, parseCSV } from "@/utils";

const users = ref<UserType[]>([]);
const saving = ref(false);
const error = ref("");
const editingUser = ref<UserType | null>(null);
const dialogVisible = ref(false);
const editForm = reactive({
  name: "",
  status: "",
  role: "",
  balance: 0,
  rate_percent: 100,
  password: "",
  models: [] as string[],
});

const activeUsers = computed(() => users.value.filter((item) => isActiveStatus(item.status)).length);
const commonModelOptions = ["gpt-4o", "gpt-4o-mini", "gpt-4.1", "gpt-4.1-mini", "claude-3-5-sonnet", "claude-3-7-sonnet", "claude-sonnet-4", "gemini-2.0-flash", "gemini-2.5-flash", "gemini-2.5-pro"];

async function load() {
  users.value = await adminAPI.users();
}

function selectUser(user: UserType) {
  editingUser.value = user;
  dialogVisible.value = true;
  editForm.name = user.name;
  editForm.status = user.status;
  editForm.role = user.role;
  editForm.balance = user.balance;
  editForm.rate_percent = user.rate_percent;
  editForm.password = "";
  editForm.models = parseAllowedModels(user.allowed_models_json);
}

function closeDialog() {
  dialogVisible.value = false;
  editingUser.value = null;
  editForm.password = "";
  editForm.models = [];
}

function parseAllowedModels(raw: string): string[] {
  const text = String(raw || "").trim();
  if (!text) return [];
  try {
    const parsed = JSON.parse(text);
    if (Array.isArray(parsed)) {
      return parsed.map((item) => String(item).trim()).filter(Boolean);
    }
  } catch {
    // fall through to csv parsing
  }
  return text
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

async function saveUser() {
  if (!editingUser.value) return;
  error.value = "";
  saving.value = true;
  try {
    await adminAPI.updateUser(editingUser.value.id, {
      name: editForm.name,
      status: editForm.status,
      role: editForm.role,
      balance: Number(editForm.balance ?? 0),
      rate_percent: Number(editForm.rate_percent ?? 100),
      password: editForm.password.trim() || undefined,
      allowed_models: editForm.models,
    });
    closeDialog();
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "保存失败";
  } finally {
    saving.value = false;
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
  grid-template-columns: 1fr 1fr 1fr 1fr;
  gap: 20px;
}

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

.stat-mini-icon.users {
  background: var(--primary-lighter);
  color: var(--primary-color);
}

.stat-mini-icon.active {
  background: var(--success-light);
  color: var(--success-color);
}

.stat-mini-icon.balance {
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

.create-section,
.users-section {
  overflow: visible;
}

.create-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 8px;
}

.form-item {
  margin-bottom: 0;
}

.form-item-wide {
  margin-bottom: 16px;
}

.submit-button {
  align-self: flex-start;
  min-width: 160px;
}

.user-count {
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

.email-cell,
.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.email-cell svg,
.name-cell svg {
  color: var(--text-muted);
}

.balance-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.models-text,
.time-text {
  font-size: 13px;
  color: var(--text-muted);
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
