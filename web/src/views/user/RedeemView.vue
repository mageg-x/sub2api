<template>
  <ElCard shadow="never">
    <template #header>
      <div style="display: flex; align-items: center; gap: 8px">
        <Gift :size="16" />
        <span>兑换码</span>
      </div>
    </template>
    <ElForm label-position="top">
      <ElFormItem label="兑换码">
        <ElInput
          v-model="code"
          placeholder="输入兑换码"
        />
      </ElFormItem>
      <ElButton
        type="primary"
        @click="submit"
      >
        立即兑换
      </ElButton>
    </ElForm>
    <ElAlert
      v-if="message"
      :title="message"
      type="success"
      :closable="false"
      style="margin-top: 16px"
    />
    <ElAlert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
      style="margin-top: 16px"
    />
  </ElCard>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { Gift } from "lucide-vue-next";
import { ElAlert, ElButton, ElCard, ElForm, ElFormItem, ElInput } from "element-plus";
import { userAPI } from "@/api/user";

const code = ref("");
const message = ref("");
const error = ref("");

async function submit() {
  message.value = "";
  error.value = "";
  try {
    await userAPI.redeem({ code: code.value });
    message.value = "兑换成功，余额已更新";
    code.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : "redeem failed";
  }
}
</script>
