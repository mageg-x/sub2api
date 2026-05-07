<template>
  <div class="login-container">
    <div class="login-left">
      <div class="brand-section">
        <div class="brand-logo">
          <ShieldCheck :size="48" />
        </div>
        <h1 class="brand-name">sub2api</h1>
        <p class="brand-tagline">管理控制台</p>
      </div>

      <div class="login-card">
        <div class="mode-switch">
          <el-button :type="setupMode ? 'default' : 'primary'" plain @click="setupMode = false">登录</el-button>
          <el-button :type="setupMode ? 'primary' : 'default'" plain @click="setupMode = true">注册</el-button>
        </div>

        <el-form ref="formRef" :model="formData" :rules="rules" @submit.prevent="setupMode ? handleBootstrap() : handleLogin()">
          <el-form-item v-if="setupMode" prop="name">
            <el-input v-model="formData.name" placeholder="管理员名称" size="large" :prefix-icon="User" />
          </el-form-item>

          <el-form-item prop="email">
            <el-input v-model="formData.email" placeholder="管理员邮箱" size="large" :prefix-icon="Mail" />
          </el-form-item>

          <el-form-item prop="password">
            <el-input v-model="formData.password" type="password" :placeholder="setupMode ? '设置管理员密码' : '密码'" size="large" show-password :prefix-icon="Lock" />
          </el-form-item>

          <el-form-item prop="adminToken">
            <el-input v-model="formData.adminToken" placeholder="管理员 Token" size="large" :prefix-icon="Key">
              <template #append>
                <el-tooltip content="首次部署时填写初始化 Token，之后仍用于访问管理接口">
                  <el-button><HelpCircle /></el-button>
                </el-tooltip>
              </template>
            </el-input>
          </el-form-item>

          <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="error-alert" />

          <el-button type="primary" native-type="submit" size="large" :loading="loading" class="login-button">
            {{ setupMode ? "注册并进入控制台" : "进入控制台" }}
          </el-button>
        </el-form>

        <div class="card-footer">
          <el-link type="info" :underline="'never'" @click="goToHome">
            <ArrowLeft :size="16" />
            返回首页
          </el-link>
        </div>
      </div>
    </div>

    <div class="login-right">
      <div class="showcase-section">
        <div class="showcase-header">
          <h2>管理控制台功能</h2>
          <p>强大的管理工具，助您轻松运营平台</p>
        </div>

        <div class="features">
          <div class="feature-item">
            <div class="feature-icon"><Users :size="24" /></div>
            <div class="feature-text">
              <h3>用户管理</h3>
              <p>管理平台注册用户</p>
            </div>
          </div>

          <div class="feature-item">
            <div class="feature-icon"><Boxes :size="24" /></div>
            <div class="feature-text">
              <h3>上游账户</h3>
              <p>管理认证与刷新中的上游账号</p>
            </div>
          </div>

          <div class="feature-item">
            <div class="feature-icon"><Banknote :size="24" /></div>
            <div class="feature-text">
              <h3>价格配置</h3>
              <p>配置模型计费规则</p>
            </div>
          </div>

          <div class="feature-item">
            <div class="feature-icon"><CreditCard :size="24" /></div>
            <div class="feature-text">
              <h3>支付订单</h3>
              <p>管理充值订单</p>
            </div>
          </div>
        </div>

        <div class="stats-row">
          <div class="stat-item">
            <div class="stat-value">OAuth</div>
            <div class="stat-label">多上游认证接入</div>
          </div>
          <div class="stat-divider"></div>
          <div class="stat-item">
            <div class="stat-value">Token</div>
            <div class="stat-label">用量计费闭环</div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from "vue";
import { useRouter } from "vue-router";
import { Mail, Lock, ShieldCheck, HelpCircle, ArrowLeft, Users, Boxes, Banknote, CreditCard, User, Key } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput, ElLink, ElTooltip } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import { adminAPI } from "@/api/admin";
import { login } from "@/api/auth";
import { apiURL } from "@/api/client";
import { clearAdminToken, saveAuth, saveAdminToken } from "@/store/session";

const router = useRouter();
const formRef = ref<FormInstance>();
const loading = ref(false);
const error = ref("");
const setupMode = ref(false);

const formData = reactive({
  name: "",
  email: "",
  password: "",
  adminToken: localStorage.getItem("sub2api_admin_token") || "",
});

const rules = computed<FormRules>(() => ({
  name: setupMode.value ? [{ required: true, message: "请输入管理员名称", trigger: "blur" }] : [],
  email: [
    { required: true, message: "请输入邮箱", trigger: "blur" },
    { type: "email", message: "请输入有效的邮箱地址", trigger: "blur" },
  ],
  password: [
    { required: true, message: "请输入密码", trigger: "blur" },
    { min: 6, message: "密码至少 6 位", trigger: "blur" },
  ],
  adminToken: [{ required: true, message: "请输入管理员 Token", trigger: "blur" }],
}));

async function handleLogin() {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (!valid) return;

    loading.value = true;
    error.value = "";

    try {
      const result = await login(formData.email, formData.password);
      saveAuth(result.access_token, result.refresh_token, result.user);
      saveAdminToken(formData.adminToken);
      await adminAPI.dashboard();
      await router.replace("/admin/dashboard");
    } catch (err) {
      clearAdminToken();
      error.value = err instanceof Error ? err.message : "登录失败";
    } finally {
      loading.value = false;
    }
  });
}

async function handleBootstrap() {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (!valid) return;

    loading.value = true;
    error.value = "";

    try {
      const res = await fetch(apiURL("/api/admin/bootstrap"), {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Admin-Token": formData.adminToken,
        },
        body: JSON.stringify({
          name: formData.name,
          email: formData.email,
          password: formData.password,
        }),
      });
      const payload = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(payload?.error || "初始化失败");
      }
      saveAuth(payload.access_token, payload.refresh_token, payload.user);
      saveAdminToken(formData.adminToken);
      await router.replace("/admin/dashboard");
    } catch (err) {
      error.value = err instanceof Error ? err.message : "初始化失败";
    } finally {
      loading.value = false;
    }
  });
}

function goToHome() {
  router.push("/");
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: grid;
  grid-template-columns: 1fr 1fr;
}

.login-left {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(160deg, var(--bg-body) 0%, var(--bg-subtle) 50%, hsl(234, 40%, 92%) 100%);
  position: relative;
  overflow: hidden;
}

.brand-section {
  text-align: center;
  margin-bottom: 16px;
}

.brand-logo {
  width: 60px;
  height: 60px;
  background: linear-gradient(135deg, var(--primary-color), var(--accent-color));
  border-radius: var(--radius-xl);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 10px;
  color: white;
  box-shadow: var(--shadow-primary);
}

.brand-name {
  font-size: 28px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 4px;
  letter-spacing: -0.02em;
}

.brand-tagline {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0;
}

.login-card {
  width: 100%;
  max-width: 380px;
  background: var(--bg-raised);
  border-radius: var(--radius-2xl);
  padding: 22px;
  box-shadow: var(--shadow-lg);
  border: 1px solid var(--border-default);
}

.mode-switch {
  display: flex;
  gap: 10px;
  margin-bottom: 18px;
}

.mode-switch :deep(.el-button) {
  flex: 1;
}

.card-header {
  margin-bottom: 16px;
  text-align: center;
}

.card-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 4px;
}

.card-subtitle {
  font-size: 12px;
  color: var(--text-muted);
  margin: 0;
}

:deep(.el-form-item) {
  margin-bottom: 22px;
}

.error-alert {
  margin-bottom: 10px;
  border-radius: var(--radius-md);
}

.login-button {
  width: 100%;
  height: 40px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  background: linear-gradient(135deg, var(--primary-color), var(--accent-color));
  border: none;
  transition:
    transform var(--transition-fast),
    box-shadow var(--transition-fast);
}

.login-button:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-primary-lg);
}

.card-footer {
  margin-top: 12px;
  text-align: center;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

:deep(.el-link) {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.login-right {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: linear-gradient(150deg, var(--primary-color), var(--accent-color));
  position: relative;
  overflow: hidden;
}

.login-right::before {
  content: "";
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 60%);
  animation: pulse 15s infinite;
}

@keyframes pulse {
  0%,
  100% {
    transform: scale(1);
    opacity: 0.5;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.3;
  }
}

.showcase-section {
  max-width: 500px;
  position: relative;
  z-index: 1;
}

.showcase-header {
  text-align: center;
  margin-bottom: 20px;
  color: white;
}

.showcase-header h2 {
  font-size: 22px;
  font-weight: 700;
  margin: 0 0 6px;
}

.showcase-header p {
  font-size: 13px;
  opacity: 0.9;
  margin: 0;
}

.features {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  margin-bottom: 18px;
}

.feature-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  background: rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(10px);
  padding: 12px 40px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.feature-icon {
  width: 34px;
  height: 34px;
  background: rgba(255, 255, 255, 0.2);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  flex-shrink: 0;
}

.feature-text h3 {
  font-size: 13px;
  font-weight: 600;
  margin: 0 0 2px;
  color: white;
}

.feature-text p {
  font-size: 11px;
  margin: 0;
  color: rgba(255, 255, 255, 0.8);
}

.stats-row {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 20px;
  background: rgba(255, 255, 255, 0.1);
  padding: 12px 20px;
  border-radius: var(--radius-md);
}

.stat-item {
  text-align: center;
}

.stat-value {
  font-size: 18px;
  font-weight: 700;
  color: white;
}

.stat-label {
  font-size: 11px;
  color: rgba(255, 255, 255, 0.8);
  margin-top: 2px;
}

.stat-divider {
  width: 1px;
  height: 28px;
  background: rgba(255, 255, 255, 0.2);
}
</style>
