<template>
  <div class="login-container">
    <div class="login-left">
      <div class="back-home">
        <el-link type="info" @click="goHome">
          <Home :size="14" />
          <span>{{ t("auth.returnHome") }}</span>
        </el-link>
      </div>

      <div class="brand-section">
        <div class="brand-logo">
          <img :src="logoUrl" alt="sub2api" />
        </div>
        <h1 class="brand-name">sub2api</h1>
        <p class="brand-tagline">{{ t("userLogin.apiPlatform") }}</p>
      </div>

      <div class="login-card">
        <div class="card-tabs">
          <button :class="['tab-button', { active: mode === 'login' }]" @click="mode = 'login'">{{ t("auth.login") }}</button>
          <button :class="['tab-button', { active: mode === 'register' }]" @click="mode = 'register'">{{ t("auth.register") }}</button>
        </div>

        <el-form v-if="mode === 'register'" ref="formRef" :model="formData" :rules="registerRules" @submit.prevent="handleSubmit">
          <el-form-item prop="name">
            <el-input v-model="formData.name" :placeholder="t('auth.username')" size="large" :prefix-icon="User" />
          </el-form-item>

          <el-form-item prop="email">
            <el-input v-model="formData.email" type="email" :placeholder="t('auth.email')" size="large" :prefix-icon="Mail" />
          </el-form-item>

          <el-form-item prop="password">
            <el-input v-model="formData.password" type="password" :placeholder="t('userLogin.passwordMinLengthPlaceholder')" size="large" show-password :prefix-icon="Lock" />
          </el-form-item>

          <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="error-alert" />

          <el-button type="primary" native-type="submit" size="large" :loading="loading" class="login-button"> {{ t("auth.createAccount") }} </el-button>
        </el-form>

        <el-form v-else ref="loginFormRef" :model="loginData" :rules="loginRules" @submit.prevent="handleLogin">
          <el-form-item prop="email">
            <el-input v-model="loginData.email" type="email" :placeholder="t('auth.email')" size="large" :prefix-icon="Mail" />
          </el-form-item>

          <el-form-item prop="password">
            <el-input v-model="loginData.password" type="password" :placeholder="t('auth.password')" size="large" show-password :prefix-icon="Lock" />
          </el-form-item>

          <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon class="error-alert" />

          <el-button type="primary" native-type="submit" size="large" :loading="loading" class="login-button"> {{ t("auth.login") }} </el-button>
        </el-form>

        <div class="card-footer"></div>
      </div>
    </div>

    <div class="login-right">
      <div class="showcase-section">
        <div class="showcase-header">
          <h2>{{ t("userLogin.quickStart") }}</h2>
          <p>{{ t("userLogin.quickStartDesc") }}</p>
        </div>

        <div class="steps">
          <div class="step">
            <div class="step-number">1</div>
            <div class="step-content">
              <h3>{{ t("userLogin.step1Title") }}</h3>
              <p>{{ t("userLogin.step1Desc") }}</p>
            </div>
          </div>

          <div class="step">
            <div class="step-number">2</div>
            <div class="step-content">
              <h3>{{ t("userLogin.step2Title") }}</h3>
              <p>{{ t("userLogin.step2Desc") }}</p>
            </div>
          </div>

          <div class="step">
            <div class="step-number">3</div>
            <div class="step-content">
              <h3>{{ t("userLogin.step3Title") }}</h3>
              <p>{{ t("userLogin.step3Desc") }}</p>
            </div>
          </div>
        </div>

        <div class="api-preview">
          <div class="preview-header">
            <span class="dot red"></span>
            <span class="dot yellow"></span>
            <span class="dot green"></span>
            <span class="preview-title">{{ t("userLogin.apiExampleTitle") }}</span>
          </div>
          <pre class="preview-code"><code><span class="keyword">curl</span> -X POST https://api.sub2api.com/v1/chat/completions \
  -H <span class="string">"Authorization: Bearer YOUR_API_KEY"</span> \
  -H <span class="string">"Content-Type: application/json"</span> \
  -d '{
    <span class="string">"model"</span>: <span class="string">"gpt-4"</span>,
    <span class="string">"messages"</span>: [
      {<span class="string">"role"</span>: <span class="string">"user"</span>, <span class="string">"content"</span>: <span class="string">"{{ t('userLogin.chatExampleMessage') }}"</span>}
    ]
  }'</code></pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Mail, Lock, User, Home } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput, ElLink } from "element-plus";
import type { FormInstance, FormRules } from "element-plus";
import logoUrl from "@/assets/logo.svg";
import { login, register } from "@/api/auth";
import { saveAuth } from "@/store/session";

const router = useRouter();
const { t } = useI18n();
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

const registerRules = computed<FormRules>(() => ({
  name: [{ required: true, message: t("auth.pleaseInputUsername"), trigger: "blur" }],
  email: [
    { required: true, message: t("auth.pleaseInputEmail"), trigger: "blur" },
    { type: "email", message: t("auth.invalidEmail"), trigger: "blur" },
  ],
  password: [
    { required: true, message: t("auth.pleaseInputPassword"), trigger: "blur" },
    { min: 6, message: t("auth.passwordMinLength"), trigger: "blur" },
  ],
}));

const loginRules = computed<FormRules>(() => ({
  email: [
    { required: true, message: t("auth.pleaseInputEmail"), trigger: "blur" },
    { type: "email", message: t("auth.invalidEmail"), trigger: "blur" },
  ],
  password: [{ required: true, message: t("auth.pleaseInputPassword"), trigger: "blur" }],
}));

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
      error.value = err instanceof Error ? err.message : t("auth.registerFailed");
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
      error.value = err instanceof Error ? err.message : t("auth.loginFailed");
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
  background: linear-gradient(160deg, var(--bg-body) 0%, var(--bg-subtle) 50%, hsl(234, 40%, 92%) 100%);
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
  max-width: 440px;
  background: var(--bg-raised);
  border-radius: var(--radius-2xl);
  padding: 22px;
  box-shadow: var(--shadow-lg);
  border: 1px solid var(--border-default);
}

.card-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  background: var(--bg-subtle);
  padding: 5px;
  border-radius: var(--radius-md);
}

.tab-button {
  flex: 1;
  padding: 8px 18px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-button.active {
  background: var(--bg-raised);
  color: var(--primary-color);
  box-shadow: var(--shadow-xs);
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
  background: radial-gradient(circle, rgba(255, 255, 255, 0.12) 0%, transparent 60%);
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
  background: rgba(255, 255, 255, 0.12);
  backdrop-filter: blur(10px);
  padding: 10px 14px;
  border-radius: var(--radius-md);
  border: 1px solid rgba(255, 255, 255, 0.18);
  transition: transform var(--transition-fast);
}

.step:hover {
  transform: translateX(6px);
}

.step-number {
  width: 30px;
  height: 30px;
  background: white;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  color: var(--primary-color);
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
  border-radius: var(--radius-md);
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
  font-size: 11px;
  color: #8b8fa8;
  margin-left: 4px;
}

.preview-code {
  margin: 0;
  padding: 14px 16px;
  font-size: 12px;
  line-height: 1.7;
  color: #cdd6f4;
  overflow-x: auto;
  font-family: var(--font-mono);
}

.keyword {
  color: #cba6f7;
}
.string {
  color: #a6e3a1;
}

@media (max-width: 768px) {
  .login-container {
    grid-template-columns: 1fr;
  }
  .login-right {
    display: none;
  }
}
</style>
