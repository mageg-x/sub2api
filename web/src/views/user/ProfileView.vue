<template>
  <div style="display: grid; gap: 18px">
    <ElAlert
      v-if="message"
      :title="message"
      type="success"
      :closable="false"
      show-icon
    />
    <ElAlert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      show-icon
    />

    <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <UserCog :size="16" />
            <span>个人资料</span>
          </div>
        </template>
        <ElForm label-position="top">
          <ElFormItem label="邮箱">
            <ElInput
              :model-value="session.user?.email || ''"
              readonly
            />
          </ElFormItem>
          <ElFormItem label="显示名称">
            <ElInput v-model="profileForm.name" />
          </ElFormItem>
          <ElButton
            type="primary"
            @click="saveProfile"
          >
            保存资料
          </ElButton>
        </ElForm>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <LockKeyhole :size="16" />
            <span>修改密码</span>
          </div>
        </template>
        <ElForm label-position="top">
          <ElFormItem label="旧密码">
            <ElInput
              v-model="passwordForm.old_password"
              type="password"
              show-password
            />
          </ElFormItem>
          <ElFormItem label="新密码">
            <ElInput
              v-model="passwordForm.new_password"
              type="password"
              show-password
            />
          </ElFormItem>
          <ElButton
            type="primary"
            @click="changePassword"
          >
            更新密码
          </ElButton>
        </ElForm>
      </ElCard>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { LockKeyhole, UserCog } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput } from "element-plus";
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
