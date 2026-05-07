<template>
  <div class="page-shell">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="sidebar-brand">
          <div class="sidebar-logo">
            <LayoutDashboard :size="20" />
          </div>
          <div>
            <h1 class="sidebar-title">sub2api</h1>
            <p class="sidebar-subtitle">用户面板</p>
          </div>
        </div>
      </div>

      <nav class="sidebar-nav">
        <div class="nav-section">
          <div class="nav-section-title">我的</div>
          <router-link v-for="item in userLinks" :key="item.to" :to="item.to" class="nav-item" :class="{ active: route.path === item.to }">
            <component :is="item.icon" class="nav-item-icon" :size="18" />
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info-mini">
          <div class="user-avatar-small">{{ userInitials }}</div>
          <div class="user-details">
            <div class="user-name-small">{{ session.user?.name || "User" }}</div>
            <div class="user-email-small">{{ session.user?.email || "" }}</div>
          </div>
        </div>
      </div>
    </aside>

    <main class="main-content">
      <header class="header">
        <div class="header-left">
          <div>
            <h2 class="page-title">{{ pageTitle }}</h2>
            <p class="page-subtitle">{{ pageSubtitle }}</p>
          </div>
        </div>

        <div class="header-right">
          <el-tag type="success" effect="dark" size="large">用户</el-tag>
          <el-tag type="warning" effect="dark" size="large">余额 {{ formatCurrency(session.user?.balance || 0) }} 元</el-tag>
          <el-dropdown @command="handleCommand">
            <div class="user-menu">
              <div class="user-avatar">{{ userInitials }}</div>
              <div class="user-info">
                <span class="user-name">{{ session.user?.name || "User" }}</span>
                <span class="user-role">{{ session.user?.role || "user" }}</span>
              </div>
              <ChevronDown :size="16" />
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <User :size="16" />
                  个人资料
                </el-dropdown-item>
                <el-dropdown-item command="switch-admin">
                  <ShieldCheck :size="16" />
                  切换到管理员
                </el-dropdown-item>
                <el-dropdown-item divided command="logout" style="color: var(--danger-color)">
                  <LogOut :size="16" />
                  退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>

      <div class="content-area">
        <router-view />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import { Bell, BookOpenText, ChevronDown, Gift, KeyRound, LayoutDashboard, LogOut, ShieldCheck, User, UserCog, WalletCards, Bolt } from "lucide-vue-next";
import { ElDropdown, ElDropdownItem, ElDropdownMenu, ElTag } from "element-plus";
import { me, logout } from "@/api/auth";
import { clearAuth, session } from "@/store/session";
import { formatCurrency } from "@/utils";

const route = useRoute();
const router = useRouter();

const userLinks = [
  { to: "/user/dashboard", label: "首页", icon: LayoutDashboard },
  { to: "/user/keys", label: "API Keys", icon: KeyRound },
  { to: "/user/usage", label: "用量记录", icon: Bolt },
  { to: "/user/payment", label: "充值", icon: WalletCards },
  { to: "/user/announcements", label: "公告", icon: Bell },
  { to: "/user/redeem", label: "兑换码", icon: Gift },
  { to: "/user/access-guide", label: "接入指南", icon: BookOpenText },
  { to: "/user/profile", label: "个人资料", icon: UserCog },
];

const pageTitle = computed(() => {
  const match = userLinks.find((item) => item.to === route.path);
  return match?.label || "sub2api";
});

const pageSubtitle = computed(() => {
  const descriptions: Record<string, string> = {
    "/user/dashboard": "您的账户概览",
    "/user/keys": "管理您的 API Keys",
    "/user/usage": "查看使用记录",
    "/user/payment": "充值余额",
    "/user/profile": "个人资料设置",
    "/user/announcements": "查看公告",
    "/user/redeem": "兑换码兑换",
    "/user/access-guide": "API 接入指南",
  };
  return descriptions[route.path] || "";
});

const userInitials = computed(() => {
  const name = session.user?.name || session.user?.email || "U";
  return name.charAt(0).toUpperCase();
});

function handleCommand(command: string) {
  switch (command) {
    case "profile":
      router.push("/user/profile");
      break;
    case "switch-admin":
      router.push("/login/admin");
      break;
    case "logout":
      void signOut();
      break;
  }
}

async function bootstrap() {
  try {
    session.user = await me();
  } catch {
    clearAuth();
    if (!route.path.startsWith("/login") && route.path !== "/") {
      router.replace("/login/user");
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
  router.replace("/");
}

onMounted(() => {
  void bootstrap();
});
</script>
