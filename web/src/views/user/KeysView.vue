<template>
  <div style="display: grid; gap: 18px">
    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <KeyRound :size="16" />
          <span>创建我的 API Key</span>
        </div>
      </template>
      <ElForm label-position="top">
        <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
          <ElFormItem label="Key 名称">
            <ElInput v-model="form.name" placeholder="如 default-client" />
          </ElFormItem>
          <ElFormItem label="允许模型">
            <ElInput v-model="form.models" placeholder="逗号分隔，可留空表示继承用户权限" />
          </ElFormItem>
        </div>
        <ElButton type="primary" @click="create">创建 Key</ElButton>
      </ElForm>
    </ElCard>

    <ElAlert v-if="lastCreatedKey" type="success" :closable="false" show-icon title="Key 已创建，可以直接复制用于客户端调用。">
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
        <span>我的 Keys</span>
      </template>
      <ElTable :data="keys" empty-text="暂无 Keys">
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
        <ElTableColumn prop="allowed_models_json" label="允许模型" min-width="240" show-overflow-tooltip />
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
import { onMounted, reactive, ref } from "vue";
import { KeyRound } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { userAPI } from "@/api/user";
import type { APIKey } from "@/api/types";
import { formatTime, isActiveStatus, maskSecret, parseCSV } from "@/utils";

const keys = ref<APIKey[]>([]);
const lastCreatedKey = ref<APIKey | null>(null);
const revealed = reactive<Record<number, boolean>>({});
const form = reactive({
  name: "",
  models: "",
});

async function load() {
  keys.value = await userAPI.keys();
}

async function create() {
  const item = await userAPI.createKey({
    name: form.name,
    allowed_models: parseCSV(form.models),
  });
  lastCreatedKey.value = item;
  revealed[item.id] = true;
  form.name = "";
  form.models = "";
  await load();
}

function copySecret(value: string) {
  void navigator.clipboard.writeText(value);
}

onMounted(() => {
  void load();
});
</script>
