<template>
  <div>
    <div v-if="message" class="success-banner">
      <el-alert :title="message" type="success" :closable="false" show-icon />
    </div>
    <div v-if="error" class="error-banner">
      <el-alert :title="error" type="error" :closable="false" show-icon />
    </div>

    <div class="profile-grid">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <UserCog :size="20" />
            个人资料
          </h3>
        </div>
        <div class="card-body">
          <el-form label-position="top" class="modern-form">
            <el-form-item label="邮箱">
              <el-input :model-value="session.user?.email || ''" readonly disabled>
                <template #prefix><Mail :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item label="显示名称">
              <el-input v-model="profileForm.name" placeholder="输入显示名称">
                <template #prefix><User :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-button type="primary" @click="saveProfile">
              <Check :size="16" style="margin-right: 6px" />
              保存资料
            </el-button>
          </el-form>
        </div>
      </div>

      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <LockKeyhole :size="20" />
            修改密码
          </h3>
        </div>
        <div class="card-body">
          <el-form label-position="top" class="modern-form">
            <el-form-item label="旧密码">
              <el-input v-model="passwordForm.old_password" type="password" show-password placeholder="输入当前密码">
                <template #prefix><Lock :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-form-item label="新密码">
              <el-input v-model="passwordForm.new_password" type="password" show-password placeholder="输入新密码">
                <template #prefix><KeyRound :size="16" /></template>
              </el-input>
            </el-form-item>
            <el-button type="primary" @click="changePassword">
              <ShieldCheck :size="16" style="margin-right: 6px" />
              更新密码
            </el-button>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { Check, KeyRound, Lock, LockKeyhole, Mail, ShieldCheck, User, UserCog } from "lucide-vue-next";
import { ElAlert, ElButton, ElForm, ElFormItem, ElInput } from "element-plus";
import { session } from "@/store/session";
import { userAPI } from "@/api/user";

const message = ref("");
const error = ref("");
const profileForm = reactive({
  name: session.user?.name || "",
});
const passwordForm = reactive({
  old_password: "",
  new_password: "",
});

async function saveProfile() {
  message.value = "";
  error.value = "";
  try {
    const user = await userAPI.updateProfile({ name: profileForm.name });
    session.user = user;
    message.value = "资料已更新";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "update failed";
  }
}

async function changePassword() {
  message.value = "";
  error.value = "";
  try {
    await userAPI.changePassword(passwordForm);
    passwordForm.old_password = "";
    passwordForm.new_password = "";
    message.value = "密码已更新";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "change password failed";
  }
}
</script>

<style scoped>
.profile-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.modern-form :deep(.el-form-item__label) {
  font-weight: 600;
  color: var(--text-secondary);
}

.success-banner,
.error-banner {
  margin-bottom: 20px;
}

@media (max-width: 768px) {
  .profile-grid {
    grid-template-columns: 1fr;
  }
}
</style>
