<template>
  <div class="auth-wrap">
    <ElCard
      class="surface-card auth-card"
      shadow="never"
    >
      <div class="eyebrow">
        <ShieldEllipsis :size="14" />
        <span>sub2api · access</span>
      </div>
      <h1
        class="hero-title"
        style="font-size: 38px"
      >
        {{ setupMode ? "初始化首个管理员" : mode === "login" ? "登录产品" : "创建账户" }}
      </h1>
      <p
        class="hero-copy"
        style="max-width: 100%"
      >
        管理员和普通用户都从这里进入。普通用户用于余额、Keys、使用量与充值；管理员额外填写 `X-Admin-Token` 后可进入管理页。
      </p>

      <div style="margin-top: 18px">
        <ElSpace
          wrap
          style="margin-bottom: 18px"
        >
          <ElButton
            :type="setupMode ? 'primary' : 'default'"
            plain
            @click="setupMode = !setupMode"
          >
            {{ setupMode ? "返回普通登录" : "首次部署？初始化管理员" }}
          </ElButton>
          <ElButton
            v-if="!setupMode"
            plain
            @click="mode = mode === 'login' ? 'register' : 'login'"
          >
            {{ mode === "login" ? "没有账号？注册" : "已有账号？登录" }}
          </ElButton>
        </ElSpace>

        <ElForm label-position="top">
          <ElFormItem
            v-if="mode === 'register' || setupMode"
            label="用户名"
          >
            <ElInput
              v-model="name"
              placeholder="用户名"
            >
              <template #prefix>
                <CircleUserRound :size="16" />
              </template>
            </ElInput>
          </ElFormItem>
          <ElFormItem label="邮箱">
            <ElInput
              v-model="email"
              type="email"
              placeholder="邮箱"
            >
              <template #prefix>
                <Mail :size="16" />
              </template>
            </ElInput>
          </ElFormItem>
          <ElFormItem label="密码">
            <ElInput
              v-model="password"
              type="password"
              show-password
              placeholder="密码，至少 6 位"
            >
              <template #prefix>
                <LockKeyhole :size="16" />
              </template>
            </ElInput>
          </ElFormItem>
          <ElFormItem label="管理员 Token">
            <ElInput
              v-model="adminToken"
              placeholder="管理员 Token（初始化管理员时必填）"
            >
              <template #prefix>
                <KeyRound :size="16" />
              </template>
            </ElInput>
          </ElFormItem>
        </ElForm>

        <ElAlert
          v-if="error"
          :title="error"
          type="error"
          show-icon
          :closable="false"
          style="margin-bottom: 16px"
        />

        <ElButton
          type="primary"
          :loading="loading"
          size="large"
          @click="setupMode ? bootstrapAdmin() : submit()"
        >
          {{ setupMode ? "初始化并进入" : mode === "login" ? "登录" : "注册并进入" }}
        </ElButton>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { CircleUserRound, KeyRound, LockKeyhole, Mail, ShieldEllipsis } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput, ElSpace } from "element-plus";
import { login, register } from "@/api/auth";
import { saveAuth, saveAdminToken } from "@/store/session";

const router = useRouter();
const mode = ref<"login" | "register">("login");
const setupMode = ref(false);
const email = ref("");
const name = ref("");
const password = ref("");
const adminToken = ref(localStorage.getItem("sub2api_admin_token") || "");
const error = ref("");
const loading = ref(false);

function defaultLanding() {
  return adminToken.value.trim() ? "/admin/dashboard" : "/user/dashboard";
}

async function submit() {
  error.value = "";
  loading.value = true;
  try {
    const result = mode.value === "login" ? await login(email.value, password.value) : await register(name.value || email.value.split("@")[0] || "user", email.value, password.value);
    saveAuth(result.access_token, result.refresh_token, result.user);
    saveAdminToken(adminToken.value);
    await router.replace(defaultLanding());
  } catch (err) {
    error.value = err instanceof Error ? err.message : "request failed";
  } finally {
    loading.value = false;
  }
}

async function bootstrapAdmin() {
  error.value = "";
  loading.value = true;
  try {
    localStorage.setItem("sub2api_admin_token", adminToken.value);
    const result = await fetch("/api/admin/bootstrap", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Admin-Token": adminToken.value,
      },
      body: JSON.stringify({
        name: name.value || email.value.split("@")[0] || "admin",
        email: email.value,
        password: password.value,
      }),
    }).then(async (res) => {
      const payload = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(payload?.error || "bootstrap failed");
      }
      return payload;
    });
    saveAuth(result.access_token, result.refresh_token, result.user);
    saveAdminToken(adminToken.value);
    await router.replace(defaultLanding());
  } catch (err) {
    error.value = err instanceof Error ? err.message : "bootstrap failed";
  } finally {
    loading.value = false;
  }
}
</script>
