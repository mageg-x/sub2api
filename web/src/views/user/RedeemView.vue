<template>
  <div class="redeem-page">
    <div v-if="message" class="success-banner">
      <el-alert :title="message" type="success" :closable="false" show-icon />
    </div>
    <div v-if="error" class="error-banner">
      <el-alert :title="error" type="error" :closable="false" show-icon />
    </div>

    <div class="redeem-container">
      <div class="surface-card redeem-card">
        <div class="card-header">
          <h3 class="card-title">
            <Gift :size="20" />
            兑换码
          </h3>
        </div>
        <div class="card-body">
          <p class="redeem-desc">输入兑换码即可将余额充值到您的账户</p>
          <el-form label-position="top" class="redeem-form">
            <el-form-item label="兑换码">
              <el-input
                v-model="code"
                placeholder="请输入兑换码"
                size="large"
                :prefix-icon="Ticket"
                clearable
                @keyup.enter="submit"
              />
            </el-form-item>
            <el-button type="primary" size="large" class="redeem-btn" :loading="loading" @click="submit">
              <Gift :size="18" style="margin-right: 8px" />
              立即兑换
            </el-button>
          </el-form>
        </div>
      </div>

      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <Info :size="20" />
            使用说明
          </h3>
        </div>
        <div class="card-body">
          <ul class="help-list">
            <li>兑换码由管理员生成并发放</li>
            <li>每个兑换码仅可使用一次</li>
            <li>兑换成功后余额即时到账</li>
            <li>如有问题请联系管理员获取帮助</li>
          </ul>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Gift, Info, Ticket } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput } from "element-plus";
import { userAPI } from "@/api/user";

const code = ref("");
const message = ref("");
const error = ref("");
const loading = ref(false);

async function submit() {
  if (!code.value.trim()) return;
  message.value = "";
  error.value = "";
  loading.value = true;
  try {
    await userAPI.redeem({ code: code.value });
    message.value = "兑换成功，余额已更新";
    code.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "redeem failed";
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.redeem-page {
  max-width: 640px;
}

.redeem-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.redeem-card {
  max-width: 480px;
}

.redeem-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0 0 20px;
  line-height: 1.6;
}

.redeem-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.redeem-btn {
  width: 100%;
  height: 44px;
  font-size: 15px;
  font-weight: 600;
}

.help-list {
  margin: 0;
  padding-left: 20px;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 2;
}

.help-list li::marker {
  color: var(--primary-color);
}

.success-banner,
.error-banner {
  margin-bottom: 20px;
}
</style>