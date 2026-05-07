<template>
  <div class="guide-page">
    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <BookOpenText :size="20" />
          接入指南
        </h3>
      </div>
      <div class="card-body">
        <p class="guide-intro">管理员先接入上游 OpenAI、Claude、Gemini、Antigravity 账户并配置价格；用户再创建自己的 API Key，通过平台代理接口发起调用。</p>
      </div>
    </div>

    <div class="provider-grid">
      <div v-for="provider in providers" :key="provider.name" class="stat-card provider-card">
        <div class="stat-header">
          <div class="stat-icon" :style="{ background: provider.color }">
            <component :is="provider.icon" :size="22" />
          </div>
        </div>
        <p class="stat-label">{{ provider.name }}</p>
        <p class="provider-endpoint mono">{{ provider.endpoint }}</p>
      </div>
    </div>

    <div class="info-grid">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <KeyRound :size="20" />
            接入信息
          </h3>
        </div>
        <div class="card-body">
          <div class="info-item">
            <span class="info-label">用户 ID</span>
            <code class="info-value mono">{{ session.user?.id || "-" }}</code>
            <el-button text size="small" @click="copyText(String(session.user?.id))">
              <Copy :size="14" />
            </el-button>
          </div>
          <div class="info-item">
            <span class="info-label">代理 Base URL</span>
            <code class="info-value mono">{{ baseURL }}</code>
            <el-button text size="small" @click="copyText(baseURL)">
              <Copy :size="14" />
            </el-button>
          </div>
          <div class="info-item">
            <span class="info-label">认证方式</span>
            <el-tag type="primary">Authorization: Bearer YOUR_API_KEY</el-tag>
          </div>
        </div>
      </div>

      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">
            <Send :size="20" />
            支持接口
          </h3>
        </div>
        <div class="card-body">
          <div class="endpoint-tags">
            <el-tag v-for="ep in endpoints" :key="ep" effect="plain" class="endpoint-tag">{{ ep }}</el-tag>
          </div>
        </div>
      </div>
    </div>

    <div class="example-grid">
      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">Chat Completions 示例</h3>
          <el-button text @click="copyText(chatExample)">
            <Copy :size="14" style="margin-right: 4px" />
            复制
          </el-button>
        </div>
        <div class="card-body">
          <pre class="code-block">{{ chatExample }}</pre>
        </div>
      </div>

      <div class="surface-card">
        <div class="card-header">
          <h3 class="card-title">Responses 示例</h3>
          <el-button text @click="copyText(responsesExample)">
            <Copy :size="14" style="margin-right: 4px" />
            复制
          </el-button>
        </div>
        <div class="card-body">
          <pre class="code-block">{{ responsesExample }}</pre>
        </div>
      </div>
    </div>

    <div class="surface-card">
      <div class="card-header">
        <h3 class="card-title">
          <Lightbulb :size="20" />
          使用建议
        </h3>
      </div>
      <div class="card-body">
        <ul class="tips-list">
          <li>到「我的 API Keys」创建专用 Key，再用于客户端调用</li>
          <li>调用成功后到「我的用量」查看 token 消耗和扣费</li>
          <li>余额不足时先到「充值」页发起 gopay 订单，回调后自动入账</li>
        </ul>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { BookOpenText, Copy, KeyRound, Lightbulb, Send, Bot, Sparkles, Hexagon, Zap } from "lucide-vue-next";
import { ElButton, ElTag } from "element-plus";
import { publicAPIOrigin } from "@/api/client";
import { session } from "@/store/session";

const baseURL = publicAPIOrigin();

function copyText(value: string) {
  void navigator.clipboard.writeText(value);
}

const providers = [
  { name: "OpenAI", endpoint: "`/v1/chat/completions` / `/v1/responses`", icon: Bot, color: "linear-gradient(135deg, #10a37f, #1a7f64)" },
  { name: "Claude", endpoint: "`/v1/messages` / `/v1/messages/count_tokens`", icon: Sparkles, color: "linear-gradient(135deg, #d97706, #b45309)" },
  { name: "Gemini", endpoint: "`/v1beta/models/*` / `/v1/models/*`", icon: Hexagon, color: "linear-gradient(135deg, #4285f4, #2563eb)" },
  { name: "Antigravity", endpoint: "`/v1internal:*` / 专用兼容入口", icon: Zap, color: "linear-gradient(135deg, #8b5cf6, #6d28d9)" },
];

const endpoints = [
  "/v1/chat/completions",
  "/v1/responses",
  "/v1/embeddings",
  "/v1/messages",
  "/v1/messages/count_tokens",
  "/v1beta/models/*",
  "/v1/models/*",
];

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

<style scoped>
.guide-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.guide-intro {
  font-size: 15px;
  color: var(--text-secondary);
  line-height: 1.8;
  margin: 0;
}

.provider-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.provider-card .stat-value {
  display: none;
}

.provider-endpoint {
  font-size: 12px !important;
  color: var(--text-muted) !important;
  margin-top: 4px;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 24px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-light);
}

.info-item:last-child {
  border-bottom: none;
}

.info-label {
  font-size: 13px;
  color: var(--text-muted);
  white-space: nowrap;
  min-width: 100px;
}

.info-value {
  flex: 1;
  font-size: 13px;
  background: var(--border-light);
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  word-break: break-all;
}

.endpoint-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.endpoint-tag {
  font-family: var(--font-mono, monospace);
  font-size: 13px;
}

.example-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 24px;
}

.code-block {
  margin: 0;
  padding: 16px;
  background: var(--border-light);
  border-radius: var(--radius-md);
  font-size: 12.5px;
  line-height: 1.6;
  overflow-x: auto;
  color: var(--text-primary);
}

.tips-list {
  margin: 0;
  padding-left: 20px;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 2.2;
}

.tips-list li::marker {
  color: var(--primary-color);
}

@media (max-width: 1024px) {
  .provider-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .info-grid,
  .example-grid {
    grid-template-columns: 1fr;
  }
}
</style>