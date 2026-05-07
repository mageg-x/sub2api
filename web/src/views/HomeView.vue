<template>
  <div class="home-page">
    <section class="hero-section">
      <h1 class="hero-title">sub2api</h1>
      <p class="hero-subtitle">企业级 AI API 分发平台</p>
      <el-button type="primary" size="large" class="cta-button" @click="goToLogin">
        立即开始
      </el-button>
    </section>

    <section class="features-section">
      <div class="features-grid">
        <div v-for="feature in features" :key="feature.title" class="feature-card">
          <div class="feature-icon">
            <component :is="feature.icon" :size="20" />
          </div>
          <h3 class="feature-title">{{ feature.title }}</h3>
          <p class="feature-desc">{{ feature.desc }}</p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import {
  KeyRound,
  ShieldCheck,
  DollarSign,
  Eye,
  Zap,
  Layers,
  Headphones
} from "lucide-vue-next";
import { ElButton } from "element-plus";

const router = useRouter();

const features = [
  {
    icon: ShieldCheck,
    title: "稳定优先",
    desc: "独家网关技术确保号池稳定，可用率达 99.5%"
  },
  {
    icon: DollarSign,
    title: "高性价比",
    desc: "按量计费透明，使用成本仅为官方十分之一"
  },
  {
    icon: Eye,
    title: "透明计费",
    desc: "实时监控用量明细，无任何隐藏扣费"
  },
  {
    icon: Zap,
    title: "简单易用",
    desc: "一键创建 API Key，快速接入各大模型服务"
  },
  {
    icon: Layers,
    title: "多渠道聚合",
    desc: "Codex 导池、Claude Max 导池、Droid 导池等多渠道聚合"
  },
  {
    icon: Headphones,
    title: "专业客服",
    desc: "专业客服团队，极速响应，解决您的任何疑问"
  }
];

function goToLogin() {
  router.push("/login/user");
}
</script>

<style scoped>
.home-page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, #f5f7fa 0%, #e8ecf1 50%, #dde4ed 100%);
}

.home-header {
  padding: 16px 32px;
  background: transparent;
}

.header-content {
  max-width: 1100px;
  margin: 0 auto;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 17px;
  font-weight: 700;
  color: #1a1a2e;
}

.hero-section {
  text-align: center;
  padding: 40px 24px 28px;
  flex-shrink: 0;
}

.hero-title {
  font-size: 52px;
  font-weight: 800;
  margin: 0 0 10px;
  letter-spacing: -0.02em;
  color: transparent;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  -webkit-background-clip: text;
  background-clip: text;
}

.hero-subtitle {
  font-size: 17px;
  color: #64748b;
  margin: 0 0 22px;
  font-weight: 400;
}

.cta-button {
  height: 42px;
  padding: 0 30px;
  font-size: 15px;
  border-radius: 21px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  font-weight: 600;
  transition: all 0.3s;
}

.cta-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(102, 126, 234, 0.35);
}

.features-section {
  max-width: 900px;
  margin: 0 auto;
  padding: 20px 24px 24px;
  flex: 1;
  display: flex;
  align-items: center;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 14px;
  width: 100%;
}

.feature-card {
  background: white;
  border-radius: 12px;
  padding: 18px 16px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.feature-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 6px 18px rgba(102, 126, 234, 0.12);
  border-color: rgba(102, 126, 234, 0.15);
}

.feature-icon {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #f0eeff 0%, #e8e4ff 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 10px;
  color: #667eea;
}

.feature-title {
  font-size: 14px;
  font-weight: 700;
  margin: 0 0 5px;
  color: #1a1a2e;
}

.feature-desc {
  font-size: 11.5px;
  line-height: 1.55;
  color: #64748b;
  margin: 0;
}

.home-footer {
  text-align: center;
  padding: 16px;
  color: #94a3b8;
  font-size: 12px;
  flex-shrink: 0;
}

@media (max-width: 800px) {
  .features-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .hero-title {
    font-size: 40px;
  }
}

@media (max-width: 550px) {
  .features-grid {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .hero-title {
    font-size: 34px;
  }

  .hero-subtitle {
    font-size: 14px;
  }

  .feature-card {
    padding: 14px 12px;
  }

  .feature-icon {
    width: 32px;
    height: 32px;
  }
}
</style>
