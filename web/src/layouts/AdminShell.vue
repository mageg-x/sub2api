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
            <p class="sidebar-subtitle">管理控制台</p>
          </div>
        </div>
      </div>

      <nav class="sidebar-nav">
        <div class="nav-section">
          <div class="nav-section-title">管理</div>
          <router-link v-for="item in adminLinks" :key="item.to" :to="item.to" class="nav-item" :class="{ active: route.path === item.to }">
            <component :is="item.icon" class="nav-item-icon" :size="18" />
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info-mini">
          <div class="user-avatar-small">{{ userInitials }}</div>
          <div class="user-details">
            <div class="user-name-small">{{ session.user?.name || "Admin" }}</div>
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
          <el-tag type="danger" effect="dark">管理员</el-tag>

          <el-dropdown @command="handleCommand">
            <div class="user-menu">
              <div class="user-avatar">{{ userInitials }}</div>
              <div class="user-info">
                <span class="user-name">{{ session.user?.name || "Admin" }}</span>
                <span class="user-role">{{ session.user?.role || "admin" }}</span>
              </div>
              <ChevronDown :size="16" />
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="switch-user">
                  <User :size="16" />
                  切换到用户
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
import { computed, onMounted, ref } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import {
  BarChart3,
  Banknote,
  Bug,
  ChevronDown,
  Gauge,
  LayoutDashboard,
  LogOut,
  Megaphone,
  ReceiptText,
  ShieldUser,
  Ticket,
  Users,
  KeyRound,
  User,
} from "lucide-vue-next";
import { ElDropdown, ElDropdownItem, ElDropdownMenu, ElTag } from "element-plus";
import { me, logout } from "@/api/auth";
import { clearAuth, session } from "@/store/session";

const route = useRoute();
const router = useRouter();

const adminLinks = [
  { to: "/admin/dashboard", label: "总览", icon: LayoutDashboard },
  { to: "/admin/users", label: "用户管理", icon: Users },
  { to: "/admin/api-keys", label: "API Keys", icon: KeyRound },
  { to: "/admin/accounts", label: "上游账户", icon: ShieldUser },
  { to: "/admin/usage", label: "用量统计", icon: BarChart3 },
  { to: "/admin/prices", label: "价格配置", icon: Banknote },
  { to: "/admin/payments", label: "支付订单", icon: ReceiptText },
  { to: "/admin/announcements", label: "公告管理", icon: Megaphone },
  { to: "/admin/coupons", label: "兑换码", icon: Ticket },
  { to: "/admin/errors", label: "错误日志", icon: Bug },
  { to: "/admin/system", label: "系统指标", icon: Gauge },
];

const pageTitle = computed(() => {
  const match = adminLinks.find((item) => item.to === route.path);
  return match?.label || "sub2api";
});

const pageSubtitle = computed(() => {
  const descriptions: Record<string, string> = {
    "/admin/dashboard": "平台整体运行状态概览",
    "/admin/users": "管理平台注册用户",
    "/admin/api-keys": "管理平台 API Keys",
    "/admin/accounts": "管理上游 OAuth 账户",
    "/admin/usage": "查看系统使用情况",
    "/admin/prices": "配置模型价格策略",
    "/admin/payments": "管理支付订单",
    "/admin/announcements": "发布系统公告",
    "/admin/coupons": "管理兑换码",
    "/admin/errors": "查看错误日志",
    "/admin/system": "系统指标监控",
  };
  return descriptions[route.path] || "";
});

const userInitials = computed(() => {
  const name = session.user?.name || session.user?.email || "A";
  return name.charAt(0).toUpperCase();
});

function handleCommand(command: string) {
  switch (command) {
    case "switch-user":
      router.push("/user/dashboard");
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
