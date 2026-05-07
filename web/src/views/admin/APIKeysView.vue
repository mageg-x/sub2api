<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">平台 Key 总数</p>
        <p class="stat-value">{{ keys.length }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">活跃 Key</p>
        <p class="stat-value">{{ activeCount }}</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">最近使用过</p>
        <p class="stat-value">
          {{ keys.filter((item) => item.last_used_at_ms > 0).length }}
        </p>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <KeyRound :size="16" />
          <span>为用户创建 API Key</span>
        </div>
      </template>
      <ElForm label-position="top">
        <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
          <ElFormItem label="用户 ID">
            <ElInput v-model.number="form.user_id" type="number" />
          </ElFormItem>
          <ElFormItem label="Key 名称">
            <ElInput v-model="form.name" />
          </ElFormItem>
          <ElFormItem label="允许模型">
            <ElInput v-model="form.models" placeholder="逗号分隔" />
          </ElFormItem>
          <ElFormItem label="过期时间戳(ms)">
            <ElInput v-model.number="form.expires_at_ms" type="number" />
          </ElFormItem>
        </div>
        <ElButton type="primary" @click="create">创建 Key</ElButton>
      </ElForm>
    </ElCard>

    <ElAlert
      v-if="lastCreatedKey"
      type="success"
      :closable="false"
      show-icon
      title="API Key 已创建，可以复制交给用户。"
    >
      <template #default>
        <div style="display: grid; gap: 10px">
          <div class="mono">{{ lastCreatedKey.secret }}</div>
          <div style="display: flex; gap: 8px; flex-wrap: wrap">
            <ElButton size="small" type="primary" @click="copySecret(lastCreatedKey.secret)">复制 Secret</ElButton>
            <ElButton size="small" @click="lastCreatedKey = null">关闭</ElButton>
          </div>
        </div>
      </template>
    </ElAlert>

    <ElCard shadow="never">
      <template #header>
        <span>平台 API Keys</span>
      </template>
      <ElTable :data="keys" empty-text="暂无 Keys">
        <ElTableColumn prop="id" label="ID" width="80" />
        <ElTableColumn prop="user_id" label="用户" width="90" />
        <ElTableColumn prop="name" label="名称" min-width="160" />
        <ElTableColumn label="Secret" min-width="260">
          <template #default="{ row }">
            <div style="display: flex; align-items: center; gap: 8px; flex-wrap: wrap">
              <span class="mono">{{ revealed[row.id] ? row.secret : maskSecret(row.secret) }}</span>
              <ElButton text size="small" @click="revealed[row.id] = !revealed[row.id]">
                {{ revealed[row.id] ? "隐藏" : "显示" }}
              </ElButton>
              <ElButton text size="small" type="primary" @click="copySecret(row.secret)">复制</ElButton>
            </div>
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="110">
          <template #default="{ row }">
            <ElTag :type="isActiveStatus(row.status) ? 'success' : 'info'">{{ row.status }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="allowed_models_json" label="允许模型" min-width="220" show-overflow-tooltip />
        <ElTableColumn label="到期时间" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.expires_at_ms) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="最后使用" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_used_at_ms) }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { KeyRound } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
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

