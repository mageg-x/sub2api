<template>
  <div>
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <KeyRound :size="24" />
          </div>
        </div>
        <p class="stat-label">平台 Key 总数</p>
        <p class="stat-value">{{ keys.length }}</p>
        <p class="stat-helper">全部 API Keys</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon active">
            <CheckCircle :size="24" />
          </div>
        </div>
        <p class="stat-label">活跃 Key</p>
        <p class="stat-value">{{ activeCount }}</p>
        <p class="stat-helper">状态正常</p>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <div class="stat-icon used">
            <Zap :size="24" />
          </div>
        </div>
        <p class="stat-label">最近使用过</p>
        <p class="stat-value">{{ keys.filter((item) => item.last_used_at_ms > 0).length }}</p>
        <p class="stat-helper">有调用记录</p>
      </div>
    </div>

    <transition name="slide-fade">
      <div v-if="lastCreatedKey" class="surface-card success-alert">
        <div class="alert-header">
          <div class="alert-icon">
            <ShieldCheck :size="20" />
          </div>
          <div class="alert-content">
            <h4>Key 创建成功</h4>
            <p>请立即复制并妥善保管，关闭后将无法再次查看完整 Secret</p>
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
            复制 Secret
          </el-button>
          <el-button @click="lastCreatedKey = null">关闭</el-button>
        </div>
      </div>
    </transition>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <KeyRound :size="20" />
          为用户创建 API Key
        </h3>
      </div>
      <div class="card-body">
        <el-form label-position="top" class="modern-form">
          <div class="form-grid">
            <el-form-item label="用户 ID">
              <el-input v-model.number="form.user_id" type="number" placeholder="输入用户 ID" />
            </el-form-item>
            <el-form-item label="Key 名称">
              <el-input v-model="form.name" placeholder="如 default-client" />
            </el-form-item>
            <el-form-item label="允许模型">
              <el-input v-model="form.models" placeholder="逗号分隔，留空则继承用户权限" />
            </el-form-item>
            <el-form-item label="过期时间戳(ms)">
              <el-input v-model.number="form.expires_at_ms" type="number" placeholder="0 表示永不过期" />
            </el-form-item>
          </div>
          <el-button type="primary" @click="create">
            <KeyRound :size="16" style="margin-right: 6px" />
            创建 Key
          </el-button>
        </el-form>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <ShieldCheck :size="20" />
          平台 API Keys
        </h3>
        <span class="key-count">{{ keys.length }} 个 Key</span>
      </div>
      <div class="card-body">
        <el-table :data="keys" empty-text="暂无 Keys" class="modern-table" :stripe="true">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="user_id" label="用户" width="70">
            <template #default="{ row }">
              <span class="user-id">#{{ row.user_id }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="name" label="名称" min-width="120">
            <template #default="{ row }">
              <div class="key-name">
                <KeyRound :size="14" />
                <span>{{ row.name }}</span>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="Secret" min-width="200">
            <template #default="{ row }">
              <div class="secret-cell">
                <code class="secret-preview mono">{{ revealed[row.id] ? row.secret : maskSecret(row.secret) }}</code>
                <el-button text size="small" @click="revealed[row.id] = !revealed[row.id]">
                  {{ revealed[row.id] ? "隐藏" : "显示" }}
                </el-button>
                <el-button text size="small" type="primary" @click="copySecret(row.secret)">
                  <Copy :size="14" />
                </el-button>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag :type="isActiveStatus(row.status) ? 'success' : 'info'" size="small">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="allowed_models_json" label="允许模型" min-width="130" show-overflow-tooltip />
          <el-table-column label="到期时间" width="140">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.expires_at_ms) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="最后使用" width="140">
            <template #default="{ row }">
              <span class="time-text">{{ formatTime(row.last_used_at_ms) }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CheckCircle, Copy, KeyRound, ShieldCheck, X, Zap } from "lucide-vue-next";
import { ElButton, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { APIKey } from "@/api/types";
import { formatTime, isActiveStatus, maskSecret, parseCSV } from "@/utils";

const keys = ref<APIKey[]>([]);
const lastCreatedKey = ref<APIKey | null>(null);
const revealed = reactive<Record<number, boolean>>({});
const form = reactive({
  user_id: 0,
  name: "",
  models: "",
  expires_at_ms: 0,
});

const activeCount = computed(() => keys.value.filter((item) => isActiveStatus(item.status)).length);

async function load() {
  keys.value = await adminAPI.keys();
}

async function create() {
  const item = await adminAPI.createKey({
    user_id: Number(form.user_id || 0),
    name: form.name,
    allowed_models: parseCSV(form.models),
    expires_at_ms: Number(form.expires_at_ms || 0),
  });
  lastCreatedKey.value = item;
  revealed[item.id] = true;
  form.user_id = 0;
  form.name = "";
  form.models = "";
  form.expires_at_ms = 0;
  await load();
}

function copySecret(value: string) {
  void navigator.clipboard.writeText(value);
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.success-alert {
  overflow: visible;
  border: 1px solid var(--success-color);
  background: linear-gradient(135deg, rgba(16, 185, 129, 0.04), rgba(16, 185, 129, 0.08));
}

.alert-header {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 16px;
}

.alert-icon {
  width: 36px;
  height: 36px;
  background: var(--success-light);
  border-radius: var(--radius-md);
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
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 4px;
}

.alert-content p {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0;
}

.secret-display {
  background: var(--border-light);
  border-radius: var(--radius-md);
  padding: 12px 16px;
  margin-bottom: 12px;
}

.secret-key {
  font-size: 13px;
  color: var(--text-primary);
  word-break: break-all;
}

.secret-actions {
  display: flex;
  gap: 10px;
}

.key-count {
  font-size: 13px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-full);
  font-weight: 500;
}

.key-name {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
}

.key-name svg {
  color: var(--text-muted);
}

.user-id {
  font-weight: 600;
  color: var(--text-secondary);
}

.secret-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.secret-preview {
  font-size: 12px;
  color: var(--text-muted);
  background: var(--border-light);
  padding: 4px 8px;
  border-radius: var(--radius-sm);
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

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
  opacity: 0;
  transform: translateY(-8px);
}
</style>