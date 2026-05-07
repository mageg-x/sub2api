<template>
  <div class="page-shell">
    <div
      class="hero-panel"
      style="margin-bottom: 18px"
    >
      <div class="eyebrow">
        <LayoutDashboard :size="14" />
        <span>sub2api · product</span>
      </div>
      <h1 class="hero-title">
        管理端与用户端一起闭环
      </h1>
      <p class="hero-copy">
        这一版不再是临时嵌页，而是独立的 Vue3 产品面。管理员负责上游账号池、OAuth、价格和支付管理； 普通用户负责登录、获取 Key、查看余额、查看用量和发起充值。
      </p>
    </div>

    <ElContainer style="gap: 20px; align-items: stretch">
      <ElAside width="290px">
        <ElCard
          class="surface-card"
          shadow="never"
        >
          <template #default>
            <div class="sidebar-brand">
              <p class="sidebar-title">
                sub2api
              </p>
              <p class="sidebar-subtitle">
                单机版 SaaS 控制台
              </p>
            </div>
            <ElScrollbar max-height="calc(100vh - 260px)">
              <div style="padding-bottom: 12px">
                <div style="padding: 0 18px">
                  <p class="section-label">
                    Admin
                  </p>
                </div>
                <ElMenu
                  :default-active="route.path"
                  router
                >
                  <ElMenuItem
                    v-for="item in adminLinks"
                    :key="item.to"
                    :index="item.to"
                  >
                    <el-icon>
                      <component
                        :is="item.icon"
                        :size="16"
                      />
                    </el-icon>
                    <span>{{ item.label }}</span>
                  </ElMenuItem>
                </ElMenu>
                <div style="padding: 12px 18px 0">
                  <p class="section-label">
                    User
                  </p>
                </div>
                <ElMenu
                  :default-active="route.path"
                  router
                >
                  <ElMenuItem
                    v-for="item in userLinks"
                    :key="item.to"
                    :index="item.to"
                  >
                    <el-icon>
                      <component
                        :is="item.icon"
                        :size="16"
                      />
                    </el-icon>
                    <span>{{ item.label }}</span>
                  </ElMenuItem>
                </ElMenu>
              </div>
            </ElScrollbar>
          </template>
        </ElCard>
      </ElAside>

      <ElMain style="padding: 0">
        <ElCard
          class="surface-card content-panel"
          shadow="never"
        >
          <div class="content-toolbar">
            <div>
              <h2 class="content-title">
                {{ pageTitle }}
              </h2>
              <p class="content-subtitle">
                所有页面都围绕真实后端接口和单二进制部署链路组织。
              </p>
            </div>
            <div style="display: flex; gap: 12px; align-items: center; flex-wrap: wrap">
              <ElTag type="info">
                {{ session.user?.role || "guest" }}
              </ElTag>
              <ElTag type="success">
                余额 {{ session.user?.balance ?? 0 }}
              </ElTag>
              <ElInput
                :model-value="session.adminToken"
                placeholder="X-Admin-Token"
                style="width: 220px"
                @change="saveAdminToken"
              />
              <ElButton
                plain
                type="danger"
                @click="signOut"
              >
                <LogOut
                  :size="16"
                  style="margin-right: 6px"
                />
                退出登录
              </ElButton>
            </div>
          </div>
          <RouterView />
        </ElCard>
      </ElMain>
    </ElContainer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import {
  Bell,
  Bolt,
  BookOpenText,
  Boxes,
  Bug,
  CircleDollarSign,
  Gauge,
  Gift,
  KeyRound,
  LayoutDashboard,
  ListOrdered,
  LogOut,
  ShieldUser,
  UserCog,
  UserRound,
  WalletCards,
  Wrench,
} from "lucide-vue-next";
import { ElAside, ElButton, ElCard, ElContainer, ElInput, ElMain, ElMenu, ElMenuItem, ElScrollbar, ElTag } from "element-plus";
import { me, logout } from "@/api/auth";
import { clearAuth, saveAdminToken, session } from "@/store/session";

const route = useRoute();
const router = useRouter();

const adminLinks = [
  { to: "/admin/dashboard", label: "总览", icon: LayoutDashboard },
  { to: "/admin/users", label: "用户", icon: ShieldUser },
  { to: "/admin/api-keys", label: "平台 API Keys", icon: KeyRound },
  { to: "/admin/accounts", label: "账户 / OAuth", icon: Boxes },
  { to: "/admin/usage", label: "用量", icon: Gauge },
  { to: "/admin/prices", label: "模型价格", icon: CircleDollarSign },
  { to: "/admin/payments", label: "支付订单", icon: ListOrdered },
  { to: "/admin/announcements", label: "公告", icon: Bell },
  { to: "/admin/coupons", label: "兑换码", icon: Gift },
  { to: "/admin/errors", label: "错误日志", icon: Bug },
  { to: "/admin/system", label: "系统指标", icon: Wrench },
];

const userLinks = [
  { to: "/user/dashboard", label: "用户首页", icon: UserRound },
  { to: "/user/keys", label: "我的 API Keys", icon: KeyRound },
  { to: "/user/usage", label: "我的用量", icon: Bolt },
  { to: "/user/payment", label: "充值", icon: WalletCards },
  { to: "/user/announcements", label: "公告", icon: Bell },
  { to: "/user/redeem", label: "兑换码", icon: Gift },
  { to: "/user/access-guide", label: "接入指南", icon: BookOpenText },
  { to: "/user/profile", label: "个人资料", icon: UserCog },
];

const pageTitle = computed(() => {
  const match = [...adminLinks, ...userLinks].find((item) => item.to === route.path);
  return match?.label || "sub2api";
});

async function bootstrap() {
  try {
    session.user = await me();
  } catch {
    clearAuth();
    if (route.path !== "/login") {
      router.replace("/login");
    }
  }
}

async function signOut() {
  const refreshToken = localStorage.getItem("sub2api_refresh_token") || "";
  try {
    await logout(refreshToken);
  } catch {
    // ignore
  }
  clearAuth();
  router.replace("/login");
}

onMounted(() => {
  void bootstrap();
});
</script>
