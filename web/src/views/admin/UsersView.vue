<template>
  <div style="display: grid; gap: 18px">
    <div class="card-grid" style="grid-template-columns: repeat(3, minmax(0, 1fr))">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">总用户数</p>
        <p class="stat-value">{{ users.length }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">当前注册用户</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">活跃用户</p>
        <p class="stat-value">{{ activeUsers }}</p>
        <p class="helper-copy" style="margin: 8px 0 0">状态为 active</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">余额总览</p>
        <p class="stat-value">
          {{ formatCurrency(users.reduce((sum, item) => sum + item.balance, 0)) }}
        </p>
        <p class="helper-copy" style="margin: 8px 0 0">单位：元</p>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <CirclePlus :size="16" />
          <span>创建用户</span>
        </div>
      </template>
      <ElAlert v-if="error" :title="error" type="error" :closable="false" show-icon style="margin-bottom: 16px" />
      <ElForm label-position="top">
        <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px">
          <ElFormItem label="邮箱">
            <ElInput v-model="form.email" placeholder="user@example.com" />
          </ElFormItem>
          <ElFormItem label="用户名">
            <ElInput v-model="form.name" placeholder="显示名称" />
          </ElFormItem>
          <ElFormItem label="密码">
            <ElInput v-model="form.password" type="password" show-password placeholder="至少 6 位" />
          </ElFormItem>
          <ElFormItem label="初始余额">
            <ElInput v-model.number="form.balance" type="number" placeholder="单位：1e-4 元" />
          </ElFormItem>
        </div>
        <ElFormItem label="允许模型">
          <ElInput v-model="form.models" placeholder="如 gpt-4o,claude-3-7-sonnet,gemini-2.5-pro" />
        </ElFormItem>
        <ElButton type="primary" :loading="submitting" @click="create">创建用户</ElButton>
      </ElForm>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <ShieldUser :size="16" />
          <span>用户列表</span>
        </div>
      </template>
      <ElTable :data="users" empty-text="暂无用户">
        <ElTableColumn prop="id" label="ID" width="80" />
        <ElTableColumn prop="email" label="邮箱" min-width="210" />
        <ElTableColumn prop="name" label="名称" min-width="140" />
        <ElTableColumn prop="role" label="角色" width="110" />
        <ElTableColumn label="状态" width="110">
          <template #default="{ row }">
            <ElTag :type="isActiveStatus(row.status) ? 'success' : 'info'">{{ row.status }}</ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="余额" min-width="120">
          <template #default="{ row }"> {{ formatCurrency(row.balance) }} 元 </template>
        </ElTableColumn>
        <ElTableColumn prop="allowed_models_json" label="允许模型" min-width="240" show-overflow-tooltip />
        <ElTableColumn label="最近登录" min-width="180">
          <template #default="{ row }">
            {{ formatTime(row.last_login_at_ms) }}
          </template>
        </ElTableColumn>
      </ElTable>
    </ElCard>
  </div>
</template>


<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CirclePlus, ShieldUser } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElTable, ElTableColumn, ElTag } from "element-plus";
import { adminAPI } from "@/api/admin";
import type { User } from "@/api/types";
import { formatCurrency, formatTime, isActiveStatus, parseCSV } from "@/utils";

const users = ref<User[]>([]);
const submitting = ref(false);
const error = ref("");
const form = reactive({
  email: "",
  name: "",
  password: "",
  balance: 0,
  models: "",
});

const activeUsers = computed(() => users.value.filter((item) => isActiveStatus(item.status)).length);

async function load() {
  users.value = await adminAPI.users();
}

async function create() {
  error.value = "";
  submitting.value = true;
  try {
    await adminAPI.createUser({
      email: form.email,
      name: form.name,
      password: form.password,
      balance: Number(form.balance || 0),
      allowed_models: parseCSV(form.models),
    });
    form.email = "";
    form.name = "";
    form.password = "";
    form.balance = 0;
    form.models = "";
    await load();
  } catch (err) {
    error.value = err instanceof Error ? err.message : "create failed";
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>

