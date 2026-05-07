<template>
  <div class="home-page">
    <!-- 背景光晕 -->
    <div class="hero-glow"></div>

    <!-- Hero 区域 -->
    <section class="hero-section">
      <div class="hero-badge">
        <span class="hero-badge-dot"></span>
        AI 订阅账户聚合平台
      </div>
      <div class="hero-logo">
        <img :src="logoUrl" alt="sub2api" />
      </div>
      <h1 class="hero-title">sub2api</h1>
      <p class="hero-subtitle">把上游 AI 订阅账户变成可分发、可计费、可运营的 API 服务</p>
      <el-button class="cta-button" @click="goToLogin"> 立即开始 </el-button>
    </section>

    <!-- 特性卡片区域 -->
    <section class="features-section">
      <div class="features-grid">
        <div v-for="feature in features" :key="feature.title" class="feature-card">
          <div class="feature-header">
            <div class="feature-icon">
              <component :is="feature.icon" :size="18" :stroke-width="2" />
            </div>
            <h3 class="feature-title">{{ feature.title }}</h3>
          </div>
          <p class="feature-desc">{{ feature.desc }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { ShieldCheck, Layers, KeyRound, Zap, DollarSign, WalletCards } from "lucide-vue-next";
import { ElButton } from "element-plus";
import logoUrl from "@/assets/logo.svg";

const router = useRouter();

const features = [
  {
    icon: ShieldCheck,
    title: "统一接口",
    desc: "支持 OpenAI、Claude、Gemini、Antigravity 账户接入，统一管理 OAuth 与静态凭据。",
  },
  {
    icon: Layers,
    title: "多渠道调度",
    desc: "同一模型可挂多组上游账户，按状态、优先级和并发限制自动分配请求。",
  },
  {
    icon: KeyRound,
    title: "简单易用",
    desc: "用户创建自己的 API Key 后即可直接接入平台代理地址，无需感知背后账户细节。",
  },
  {
    icon: Zap,
    title: "稳定优先",
    desc: "高质量IP，高质量号池，可用率达 99.9%。",
  },
  {
    icon: DollarSign,
    title: "计费透明",
    desc: "按模型价格记录输入输出 token 消耗，生成用量日志并自动扣减用户余额。",
  },
  {
    icon: WalletCards,
    title: "支付便捷",
    desc: "支持微信和支付宝充值，并提供公告、优惠码、错误日志、账户与价格管理页面。",
  },
];

function goToLogin() {
  router.push("/login/user");
}
</script>

<style scoped>
.home-page {
  position: relative;
  height: 100vh;
  max-height: 1080px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: linear-gradient(175deg, #fafbfe 0%, #f2f4fc 20%, #edeff9 45%, #f4f5fb 70%, #fafbfe 100%);
  overflow: hidden;
  isolation: isolate;
}

/* ============ 背景光斑 ============ */
.home-page::before {
  content: "";
  position: absolute;
  top: -6%;
  right: -3%;
  width: 42vw;
  height: 42vw;
  max-width: 520px;
  max-height: 520px;
  background: radial-gradient(
    circle at 55% 35%,
    hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.13) 0%,
    hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.07) 30%,
    hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.03) 55%,
    transparent 72%
  );
  border-radius: 50%;
  pointer-events: none;
  z-index: 0;
  animation: orbDriftA 11s ease-in-out infinite;
  filter: blur(2px);
}

.home-page::after {
  content: "";
  position: absolute;
  bottom: -5%;
  left: -4%;
  width: 36vw;
  height: 36vw;
  max-width: 420px;
  max-height: 420px;
  background: radial-gradient(
    circle at 40% 60%,
    hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.11) 0%,
    hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.06) 28%,
    hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.02) 50%,
    transparent 70%
  );
  border-radius: 50%;
  pointer-events: none;
  z-index: 0;
  animation: orbDriftB 14s ease-in-out infinite;
  filter: blur(3px);
}

/* 第三光斑 */
.hero-glow {
  position: absolute;
  top: 22%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 55vw;
  height: 28vh;
  max-width: 700px;
  max-height: 240px;
  background: radial-gradient(
    ellipse at center,
    hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.07) 0%,
    hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.04) 35%,
    transparent 68%
  );
  border-radius: 50%;
  pointer-events: none;
  z-index: 0;
}

/* ============ 动画 ============ */
@keyframes orbDriftA {
  0%,
  100% {
    transform: translate(0, 0) scale(1);
  }
  25% {
    transform: translate(18px, -14px) scale(1.06);
  }
  50% {
    transform: translate(-8px, -22px) scale(0.94);
  }
  75% {
    transform: translate(-16px, 6px) scale(1.04);
  }
}

@keyframes orbDriftB {
  0%,
  100% {
    transform: translate(0, 0) scale(1);
  }
  30% {
    transform: translate(-14px, -18px) scale(1.07);
  }
  55% {
    transform: translate(10px, 12px) scale(0.93);
  }
  80% {
    transform: translate(-6px, 20px) scale(1.03);
  }
}

@keyframes subtlePulse {
  0%,
  100% {
    opacity: 0.7;
  }
  50% {
    opacity: 1;
  }
}

@keyframes cardShimmer {
  0% {
    background-position: -200% center;
  }
  100% {
    background-position: 200% center;
  }
}

/* ============ Hero 区域 ============ */
.hero-section {
  position: relative;
  z-index: 2;
  text-align: center;
  padding: 0 24px;
  flex-shrink: 0;
  margin-bottom: clamp(10px, 2vh, 20px);
}

.hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 6px 16px;
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.15);
  border-radius: 24px;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--primary-color);
  letter-spacing: 0.03em;
  margin-bottom: 16px;
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
  animation: subtlePulse 3.5s ease-in-out infinite;
}

.hero-badge-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--accent-color);
  box-shadow: 0 0 8px hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.5);
}

.hero-logo {
  width: 86px;
  height: 86px;
  margin: 0 auto 14px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.12);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
}

.hero-logo img {
  width: 56px;
  height: 56px;
  display: block;
  object-fit: contain;
}

.hero-title {
  font-size: clamp(38px, 5.5vw, 58px);
  font-weight: 820;
  margin: 0 0 8px;
  letter-spacing: -0.025em;
  line-height: 1.1;
  color: transparent;
  background: linear-gradient(140deg, var(--primary-color) 0%, var(--accent-color) 22%, hsl(var(--accent-h), var(--accent-s), 62%) 48%, hsl(var(--accent-h), 70%, 58%) 68%, var(--primary-color) 100%);
  background-size: 180% 180%;
  -webkit-background-clip: text;
  background-clip: text;
  animation: cardShimmer 5s ease-in-out infinite;
  filter: drop-shadow(0 2px 6px hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.18));
}

.hero-subtitle {
  font-size: clamp(14px, 1.55vw, 17px);
  color: var(--text-secondary);
  margin: 0 0 clamp(16px, 2.8vh, 24px);
  font-weight: 420;
  line-height: 1.5;
  letter-spacing: 0.01em;
  max-width: 520px;
  margin-left: auto;
  margin-right: auto;
}

.cta-button {
  height: 44px;
  padding: 0 34px;
  font-size: 15px;
  font-weight: 640;
  letter-spacing: 0.02em;
  border-radius: 24px;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--accent-color) 55%, hsl(var(--accent-h), var(--accent-s), 62%) 100%);
  background-size: 150% 150%;
  border: none;
  color: #fff;
  cursor: pointer;
  transition: all var(--transition-normal);
  box-shadow: var(--shadow-primary), var(--shadow-xs);
  animation: cardShimmer 4.5s ease-in-out infinite;
}

.cta-button:hover {
  transform: translateY(-3px);
  box-shadow:
    var(--shadow-primary-lg),
    0 3px 8px hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.18);
  filter: brightness(1.06);
}

.cta-button:active {
  transform: translateY(-1px);
  box-shadow: var(--shadow-primary);
  filter: brightness(0.97);
  transition: all 0.1s ease;
}

/* ============ 特性卡片 ============ */
.features-section {
  position: relative;
  z-index: 2;
  max-width: 920px;
  width: 100%;
  padding: 0 24px;
  flex-shrink: 0;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: clamp(10px, 1.6vw, 16px);
  width: 100%;
}

.feature-card {
  position: relative;
  background: rgba(255, 255, 255, 0.82);
  border-radius: var(--radius-lg);
  padding: clamp(14px, 2vh, 20px) clamp(12px, 1.5vw, 18px);
  cursor: default;
  transition: all var(--transition-normal);
  border: 1px solid hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.07);
  box-shadow:
    var(--shadow-sm),
    inset 0 0 0 1px rgba(255, 255, 255, 0.6);
  backdrop-filter: blur(8px);
  -webkit-backdrop-filter: blur(8px);
  overflow: hidden;
}

/* 卡片顶部微光条 */
.feature-card::before {
  content: "";
  position: absolute;
  top: 0;
  left: 12px;
  right: 12px;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.18) 25%,
    hsla(var(--accent-h), var(--accent-s), var(--accent-l), 0.22) 50%,
    hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.18) 75%,
    transparent 100%
  );
  opacity: 0;
  transition: opacity var(--transition-smooth);
  pointer-events: none;
  z-index: 1;
}

.feature-card:hover::before {
  opacity: 1;
}

/* 卡片 hover 微光扫过 */
.feature-card::after {
  content: "";
  position: absolute;
  top: -50%;
  left: -50%;
  width: 200%;
  height: 200%;
  background: radial-gradient(circle at center, rgba(255, 255, 255, 0.5) 0%, transparent 60%);
  opacity: 0;
  transform: scale(0.6);
  transition: all 0.45s ease;
  pointer-events: none;
  z-index: 0;
}

.feature-card:hover::after {
  opacity: 0.35;
  transform: scale(1);
}

.feature-card:hover {
  transform: translateY(-4px);
  box-shadow:
    var(--shadow-lg),
    inset 0 0 0 1px hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.14);
  border-color: hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.18);
  background: rgba(255, 255, 255, 0.92);
}

.feature-card:active {
  transform: translateY(-1px);
  transition: all 0.08s ease;
}

.feature-header {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  gap: 9px;
  margin-bottom: clamp(10px, 1.6vh, 14px);
}

.feature-icon {
  width: clamp(28px, 3.2vw, 32px);
  height: clamp(28px, 3.2vw, 32px);
  min-width: clamp(28px, 3.2vw, 32px);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  background: linear-gradient(135deg, var(--primary-color) 0%, var(--accent-color) 55%, hsl(var(--accent-h), var(--accent-s), 62%) 100%);
  box-shadow: var(--shadow-primary);
  transition: all var(--transition-normal);
  position: relative;
}

.feature-icon::after {
  content: "";
  position: absolute;
  inset: -3px;
  border-radius: 12px;
  background: transparent;
  box-shadow: 0 0 0 0 hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0);
  transition: all var(--transition-normal);
  pointer-events: none;
  z-index: -1;
}

.feature-card:hover .feature-icon {
  box-shadow: var(--shadow-primary-lg);
  transform: scale(1.04);
}

.feature-card:hover .feature-icon::after {
  box-shadow: 0 0 0 6px hsla(var(--primary-h), var(--primary-s), var(--primary-l), 0.08);
}

.feature-title {
  font-size: clamp(12.5px, 1.3vw, 14px);
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.01em;
  line-height: 1.2;
}

.feature-desc {
  position: relative;
  z-index: 1;
  font-size: clamp(11px, 1.05vw, 12.5px);
  line-height: 1.55;
  color: var(--text-secondary);
  margin: 0;
  letter-spacing: 0.01em;
}

/* ============ 响应式 ============ */
@media (max-width: 860px) {
  .features-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .hero-title {
    font-size: 40px;
  }

  .hero-subtitle {
    font-size: 15px;
    max-width: 420px;
  }

  .feature-card {
    padding: 16px 14px;
  }

  .home-page::before {
    width: 50vw;
    height: 50vw;
    top: -8%;
    right: -8%;
  }

  .home-page::after {
    width: 40vw;
    height: 40vw;
    bottom: -6%;
    left: -6%;
  }
}

@media (max-width: 550px) {
  .home-page {
    min-height: 720px;
    justify-content: flex-start;
    padding-top: 6vh;
  }

  .features-grid {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .hero-section {
    margin-bottom: 12px;
  }

  .hero-logo {
    width: 74px;
    height: 74px;
    margin-bottom: 12px;
  }

  .hero-logo img {
    width: 48px;
    height: 48px;
  }

  .hero-title {
    font-size: 33px;
  }

  .hero-subtitle {
    font-size: 13.5px;
    max-width: 340px;
    margin-bottom: 16px;
  }

  .hero-badge {
    font-size: 11px;
    padding: 5px 12px;
    margin-bottom: 12px;
  }

  .cta-button {
    height: 40px;
    padding: 0 26px;
    font-size: 14px;
    border-radius: 22px;
  }

  .feature-card {
    padding: 13px 11px;
    border-radius: 12px;
  }

  .feature-icon {
    width: 30px;
    height: 30px;
    min-width: 30px;
    border-radius: 8px;
  }

  .feature-title {
    font-size: 12px;
  }

  .feature-desc {
    font-size: 10.5px;
    line-height: 1.45;
  }

  .home-page::before {
    width: 60vw;
    height: 60vw;
    top: -10%;
    right: -15%;
    filter: blur(4px);
  }

  .home-page::after {
    width: 50vw;
    height: 50vw;
    bottom: -8%;
    left: -12%;
    filter: blur(4px);
  }

  .hero-glow {
    width: 70vw;
    height: 20vh;
  }
}

@media (max-width: 380px) {
  .features-grid {
    grid-template-columns: 1fr;
    gap: 9px;
    max-width: 300px;
    margin: 0 auto;
  }

  .hero-title {
    font-size: 28px;
  }

  .hero-subtitle {
    font-size: 12.5px;
    max-width: 280px;
  }

  .feature-card {
    padding: 12px 14px;
  }
}
</style>
