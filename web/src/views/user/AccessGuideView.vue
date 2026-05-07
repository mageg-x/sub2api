<template>
  <div style="display: grid; gap: 18px">
    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 8px">
          <BookOpenText :size="16" />
          <span>接入指南</span>
        </div>
      </template>
      <p class="helper-copy" style="margin: 0; line-height: 1.8">管理员先接入上游 OpenAI、Claude、Gemini、Antigravity 账户并配置价格；用户再创建自己的 API Key，通过平台代理接口发起调用。</p>
    </ElCard>

    <div style="display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px">
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">OpenAI</p>
        <p class="helper-copy" style="margin: 8px 0 0">`/v1/chat/completions` / `/v1/responses`</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">Claude</p>
        <p class="helper-copy" style="margin: 8px 0 0">`/v1/messages` / `/v1/messages/count_tokens`</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">Gemini</p>
        <p class="helper-copy" style="margin: 8px 0 0">`/v1beta/models/*` / `/v1/models/*`</p>
      </ElCard>
      <ElCard shadow="never" class="stat-card">
        <p class="stat-label">Antigravity</p>
        <p class="helper-copy" style="margin: 8px 0 0">`/v1internal:*` / 专用兼容入口</p>
      </ElCard>
    </div>

    <div style="display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px">
      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <KeyRound :size="16" />
            <span>接入信息</span>
          </div>
        </template>
        <div style="display: grid; gap: 12px">
          <div>
            <p class="stat-label">用户 ID</p>
            <div class="mono">{{ session.user?.id || "-" }}</div>
          </div>
          <div>
            <p class="stat-label">代理 Base URL</p>
            <div class="mono">{{ baseURL }}</div>
          </div>
          <div>
            <p class="stat-label">认证方式</p>
            <ElTag>Authorization: Bearer YOUR_API_KEY</ElTag>
          </div>
        </div>
      </ElCard>

      <ElCard shadow="never">
        <template #header>
          <div style="display: flex; align-items: center; gap: 8px">
            <Send :size="16" />
            <span>支持接口</span>
          </div>
        </template>
        <div style="display: grid; gap: 10px">
          <ElTag>/v1/chat/completions</ElTag>
          <ElTag>/v1/responses</ElTag>
          <ElTag>/v1/embeddings</ElTag>
          <ElTag>/v1/messages</ElTag>
          <ElTag>/v1/messages/count_tokens</ElTag>
          <ElTag>/v1beta/models/*</ElTag>
          <ElTag>/v1/models/*</ElTag>
        </div>
      </ElCard>
    </div>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between; gap: 12px">
          <span>Chat Completions 示例</span>
          <ElButton text @click="copyText(chatExample)">
            <Copy :size="14" style="margin-right: 6px" />
            复制
          </ElButton>
        </div>
      </template>
      <div class="pre-box">{{ chatExample }}</div>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; justify-content: space-between; gap: 12px">
          <span>Responses 示例</span>
          <ElButton text @click="copyText(responsesExample)">
            <Copy :size="14" style="margin-right: 6px" />
            复制
          </ElButton>
        </div>
      </template>
      <div class="pre-box">{{ responsesExample }}</div>
    </ElCard>

    <ElCard shadow="never">
      <template #header>
        <span>使用建议</span>
      </template>
      <div style="display: grid; gap: 10px">
        <p class="helper-copy" style="margin: 0">1. 到“我的 API Keys”创建专用 Key，再用于客户端调用。</p>
        <p class="helper-copy" style="margin: 0">2. 调用成功后到“我的用量”查看 token 消耗和扣费。</p>
        <p class="helper-copy" style="margin: 0">3. 余额不足时先到“充值”页发起 gopay 订单，回调后自动入账。</p>
      </div>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
import { BookOpenText, Copy, KeyRound, Send } from "lucide-vue-next";
import { ElButton, ElCard, ElTag } from "element-plus";
import { publicAPIOrigin } from "@/api/client";
import { session } from "@/store/session";

const baseURL = publicAPIOrigin();

function copyText(value: string) {
  void navigator.clipboard.writeText(value);
}

const chatExample = `curl ${baseURL}/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role":"user","content":"你好"}]
  }'`;

const responsesExample = `curl ${baseURL}/v1/responses \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4.1-mini",
    "input": "写一个摘要"
  }'`;
</script>
