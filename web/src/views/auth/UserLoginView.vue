<template>
  <div class="login-container">
    <div class="login-left">
      <div class="back-home">
        <el-link type="info" @click="goHome">
          <Home :size="14" />
          <span>返回首页</span>
        </el-link>
      </div>

      <div class="brand-section">
        <div class="brand-logo">
          <KeyRound :size="48" />
        </div>
        <h1 class="brand-name">sub2api</h1>
        <p class="brand-tagline">API 访问平台</p>
      </div>

      <div class="login-card">
        <div class="card-tabs">
          <button :class="['tab-button', { active: mode === 'login' }]" @click="mode = 'login'">登录</button>
          <button :class="['tab-button', { active: mode === 'register' }]" @click="mode = 'register'">注册</button>
        </div>


        <el-form v-if="mode === 'register'" ref="formRef" :model="formData" :rules="registerRules" @submit.prevent="handleSubmit">
          <el-form-item prop="name">
            <el-input v-model="formData.name" placeholder="用户名" size="large" :prefix-icon="User" />
          </el-form-item>

          <el-form-item prop="email">
            <el-input v-model="formData.email" type="email" placeholder="邮箱" size="large" :prefix-icon="Mail" />
          </el-form-item>

          <el-form-item prop="password">
            <el-input v-model="formData.password" type="password" placeholder="密码（至少 6 位）" size="large" show-password :prefix-icon="Lock" />
          </el-form-item>

          <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="error-alert" />

          <el-button type="primary" native-type="submit" size="large" :loading="loading" class="login-button"> 创建账户 </el-button>
        </el-form>

        <el-form v-else ref="loginFormRef" :model="loginData" :rules="loginRules" @submit.prevent="handleLogin">
          <el-form-item prop="email">
            <el-input v-model="loginData.email" type="email" placeholder="邮箱" size="large" :prefix-icon="Mail" />
          </el-form-item>

          <el-form-item prop="password">
            <el-input v-model="loginData.password" type="password" placeholder="密码" size="large" show-password :prefix-icon="Lock" />
          </el-form-item>

          <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="error-alert" />

          <el-button type="primary" native-type="submit" size="large" :loading="loading" class="login-button"> 登录 </el-button>
        </el-form>

        <div class="card-footer">
        </div>
      </div>
    </div>

    <div class="login-right">
      <div class="showcase-section">
        <div class="showcase-header">
          <h2>快速开始</h2>
          <p>三步轻松接入 AI 能力</p>
        </div>

        <div class="steps">
          <div class="step">
            <div class="step-number">1</div>
            <div class="step-content">
              <h3>注册账户</h3>
              <p>创建账户并完成充值</p>
            </div>
          </div>

          <div class="step">
            <div class="step-number">2</div>
            <div class="step-content">
              <h3>获取 API Key</h3>
              <p>在控制台创建您的专属密钥</p>
            </div>
          </div>

          <div class="step">
            <div class="step-number">3</div>
            <div class="step-content">
              <h3>开始调用</h3>
              <p>通过 API 调用各种 AI 模型</p>
            </div>
          </div>
        </div>

        <div class="api-preview">
          <div class="preview-header">
            <span class="dot red"></span>
            <span class="dot yellow"></span>
            <span class="dot green"></span>
            <span class="preview-title">API 调用示例</span>
          </div>
          <pre class="preview-code"><code><span class="keyword">curl</span> -X POST https://api.sub2api.com/v1/chat/completions \
  -H <span class="string">"Authorization: Bearer YOUR_API_KEY"</span> \
  -H <span class="string">"Content-Type: application/json"</span> \
  -d '{
    <span class="string">"model"</span>: <span class="string">"gpt-4"</span>,
    <span class="string">"messages"</span>: [
      {<span class="string">"role"</span>: <span class="string">"user"</span>, <span class="string">"content"</span>: <span class="string">"你好！"</span>}
    ]
  }'</code></pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from "vue";
import { useRouter } from "vue-router";
import { KeyRound, Mail, Lock, User, ShieldCheck, Home } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput, ElLink } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import { login, register } from "@/api/auth";
import { saveAuth, saveAdminToken } from "@/store/session";

const router = useRouter();
const mode = ref<"login" | "register">("login");
const loading = ref(false);
const error = ref("");
const formRef = ref<FormInstance>();
const loginFormRef = ref<FormInstance>();

const formData = reactive({
  name: "",
  email: "",
  password: "",
});

const loginData = reactive({
  email: "",
  password: "",
});

const registerRules: FormRules = {
  name: [{ required: true, message: "请输入用户名", trigger: "blur" }],
  email: [
    { required: true, message: "请输入邮箱", trigger: "blur" },
    { type: "email", message: "请输入有效的邮箱地址", trigger: "blur" },
  ],
  password: [
    { required: true, message: "请输入密码", trigger: "blur" },
    { min: 6, message: "密码至少 6 位", trigger: "blur" },
  ],
};

const loginRules: FormRules = {
  email: [
    { required: true, message: "请输入邮箱", trigger: "blur" },
    { type: "email", message: "请输入有效的邮箱地址", trigger: "blur" },
  ],
  password: [{ required: true, message: "请输入密码", trigger: "blur" }],
};

watch(mode, () => {
  error.value = "";
});

function goHome() {
  router.push("/");
}

async function handleSubmit() {
  if (!formRef.value) return;

  await formRef.value.validate(async (valid) => {
    if (!valid) return;

    loading.value = true;
    error.value = "";

    try {
      const result = await register(formData.name, formData.email, formData.password);
      saveAuth(result.access_token, result.refresh_token, result.user);
      await router.replace("/user/dashboard");
    } catch (err) {
      error.value = err instanceof Error ? err.message : "注册失败";
    } finally {
      loading.value = false;
    }
  });
}

async function handleLogin() {
  if (!loginFormRef.value) return;

  await loginFormRef.value.validate(async (valid) => {
    if (!valid) return;

    loading.value = true;
    error.value = "";

    try {
      const result = await login(loginData.email, loginData.password);
      saveAuth(result.access_token, result.refresh_token, result.user);
      await router.replace("/user/dashboard");
    } catch (err) {
      error.value = err instanceof Error ? err.message : "登录失败";
    } finally {
      loading.value = false;
    }
  });
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
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  position: relative;
  overflow: hidden;
}

.back-home {
  position: absolute;
  top: 20px;
  left: 24px;
  z-index: 10;
}

:deep(.back-home .el-link) {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 13px;
}

.brand-section {
  text-align: center;
  margin-bottom: 16px;
}

.brand-logo {
  width: 60px;
  height: 60px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 10px;
  color: white;
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.3);
}

.brand-name {
  font-size: 28px;
  font-weight: 700;
  color: #1a1a2e;
  margin: 0 0 4px;
  letter-spacing: -0.02em;
}

.brand-tagline {
  font-size: 13px;
  color: #64748b;
  margin: 0;
}

.login-card {
  width: 100%;
  max-width: 440px;
  background: white;
  border-radius: 16px;
  padding: 22px;
  box-shadow: 0 16px 32px rgba(0, 0, 0, 0.08);
}

.card-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  background: #f1f5f9;
  padding: 5px;
  border-radius: 10px;
}

.tab-button {
  flex: 1;
  padding: 8px 18px;
  border: none;
  background: transparent;
  border-radius: 7px;
  font-size: 13px;
  font-weight: 500;
  color: #64748b;
  cursor: pointer;
  transition: all 0.3s;
}

.tab-button.active {
  background: white;
  color: #667eea;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

:deep(.el-form-item) {
  margin-bottom: 22px;
}

:deep(.el-input__wrapper) {
  padding: 8px 12px;
  border-radius: 8px;
  box-shadow: 0 0 0 1px #e2e8f0;
  transition: all 0.3s;
}

:deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #cbd5e1;
}

:deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 2px #667eea;
}

.error-alert {
  margin-bottom: 10px;
  border-radius: 8px;
}

.login-button {
  width: 100%;
  height: 40px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  transition: transform 0.2s, box-shadow 0.2s;
}

.login-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 12px rgba(102, 126, 234, 0.3);
}

.card-footer {
  margin-top: 12px;
  text-align: center;
  padding-top: 12px;
  border-top: 1px solid #e2e8f0;
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
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
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
  background: radial-gradient(circle, rgba(255, 255, 255, 0.15) 0%, transparent 60%);
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
  margin-bottom: 18px;
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

.steps {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.step {
  display: flex;
  align-items: center;
  gap: 12px;
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(10px);
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  transition: transform 0.3s;
}

.step:hover {
  transform: translateX(6px);
}

.step-number {
  width: 30px;
  height: 30px;
  background: white;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  color: #667eea;
  flex-shrink: 0;
}

.step-content h3 {
  font-size: 13px;
  font-weight: 600;
  margin: 0 0 2px;
  color: white;
}

.step-content p {
  font-size: 11px;
  margin: 0;
  color: rgba(255, 255, 255, 0.8);
}

.api-preview {
  background: #1e1e2e;
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.25);
}

.preview-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #2e2e3e;
  border-bottom: 1px solid #3e3e4e;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot.red {
  background: #ff5f56;
}
.dot.yellow {
  background: #ffbd2e;
}
.dot.green {
  background: #27c93f;
}

.preview-title {
  margin-left: 8px;
  font-size: 11px;
  color: #888;
}

.preview-code {
  margin: 0;
  padding: 10px;
  font-size: 11px;
  line-height: 1.4;
  color: #a6accd;
  overflow-x: auto;
}

.keyword {
  color: #c678dd;
}

.string {
  color: #98c379;
}

@media (max-width: 1024px) {
  .login-container {
    grid-template-columns: 1fr;
  }

  .login-right {
    display: none;
  }

  .login-left {
    padding: 24px;
  }
}
</style>
