<template>
  <div class="page-shell">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="sidebar-brand">
          <div class="sidebar-logo">
            <LayoutDashboard :size="24" />
          </div>
          <div>
            <h1 class="sidebar-title">sub2api</h1>
            <p class="sidebar-subtitle">{{ isAdmin ? "管理控制台" : "用户面板" }}</p>
          </div>
        </div>
      </div>

      <nav class="sidebar-nav">
        <div v-if="isAdmin" class="nav-section">
          <div class="nav-section-title">管理</div>
          <router-link v-for="item in adminLinks" :key="item.to" :to="item.to" class="nav-item" :class="{ active: route.path === item.to }">
            <component :is="item.icon" class="nav-item-icon" :size="20" />
            <span>{{ item.label }}</span>
          </router-link>
        </div>

        <div class="nav-section">
          <div class="nav-section-title">{{ isAdmin ? "用户" : "我的" }}</div>
          <router-link v-for="item in userLinks" :key="item.to" :to="item.to" class="nav-item" :class="{ active: route.path === item.to }">
            <component :is="item.icon" class="nav-item-icon" :size="20" />
            <span>{{ item.label }}</span>
          </router-link>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info-mini">
          <div class="user-avatar-small">
            {{ userInitials }}
          </div>
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
          <el-tag v-if="session.user?.role" :type="isAdmin ? 'danger' : 'success'" effect="dark" size="large">
            {{ isAdmin ? "管理员" : "用户" }}
          </el-tag>

          <el-tag v-if="!isAdmin" type="warning" effect="dark" size="large"> 余额 {{ formatCurrency(session.user?.balance || 0) }} 元 </el-tag>

          <div v-if="isAdmin" class="admin-token-input">
            <el-input v-model="adminTokenInput" placeholder="X-Admin-Token" size="default" clearable @change="handleAdminTokenChange">
              <template #prefix>
                <Key :size="16" />
              </template>
            </el-input>
          </div>

          <el-dropdown @command="handleCommand">
            <div class="user-menu">
              <div class="user-avatar">
                {{ userInitials }}
              </div>
              <div class="user-info">
                <span class="user-name">{{ session.user?.name || "User" }}</span>
                <span class="user-role">{{ session.user?.role || "guest" }}</span>
              </div>
              <ChevronDown :size="16" />
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <User :size="16" />
                  个人资料
                </el-dropdown-item>
                <el-dropdown-item v-if="!isAdmin" command="switch-admin">
                  <ShieldCheck :size="16" />
                  切换到管理员
                </el-dropdown-item>
                <el-dropdown-item v-if="isAdmin" command="switch-user">
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
  Bell,
  Bolt,
  BookOpenText,
  Boxes,
  Bug,
  ChevronDown,
  CircleDollarSign,
  Gauge,
  Gift,
  Key,
  KeyRound,
  LayoutDashboard,
  ListOrdered,
  LogOut,
  Settings,
  ShieldCheck,
  ShieldUser,
  User,
  UserCog,
  UserRound,
  WalletCards,
  Wrench,
} from "lucide-vue-next";
import { ElDropdown, ElDropdownItem, ElDropdownMenu, ElInput, ElTag } from "element-plus";
import { me, logout } from "@/api/auth";
import { clearAuth, saveAdminToken, session } from "@/store/session";
import { formatCurrency } from "@/utils";

const route = useRoute();
const router = useRouter();

const adminTokenInput = ref(session.adminToken || "");

const isAdmin = computed(() => {
  return route.path.startsWith("/admin");
});

const userInitials = computed(() => {
  const name = session.user?.name || session.user?.email || "U";
  return name.charAt(0).toUpperCase();
});

const pageTitle = computed(() => {
  const match = [...adminLinks, ...userLinks].find((item) => item.to === route.path);
  return match?.label || "sub2api";
});

const pageSubtitle = computed(() => {
  const adminDescriptions: Record<string, string> = {
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

  const userDescriptions: Record<string, string> = {
    "/user/dashboard": "您的账户概览",
    "/user/keys": "管理您的 API Keys",
    "/user/usage": "查看使用记录",
    "/user/payment": "充值余额",
    "/user/profile": "个人资料设置",
    "/user/announcements": "查看公告",
    "/user/redeem": "兑换码兑换",
    "/user/access-guide": "API 接入指南",
  };

  return isAdmin.value ? adminDescriptions[route.path] || "" : userDescriptions[route.path] || "";
});

const adminLinks = [
  { to: "/admin/dashboard", label: "总览", icon: LayoutDashboard },
  { to: "/admin/users", label: "用户管理", icon: ShieldUser },
  { to: "/admin/api-keys", label: "API Keys", icon: KeyRound },
  { to: "/admin/accounts", label: "上游账户", icon: Boxes },
  { to: "/admin/usage", label: "用量统计", icon: Gauge },
  { to: "/admin/prices", label: "价格配置", icon: CircleDollarSign },
  { to: "/admin/payments", label: "支付订单", icon: ListOrdered },
  { to: "/admin/announcements", label: "公告管理", icon: Bell },
  { to: "/admin/coupons", label: "兑换码", icon: Gift },
  { to: "/admin/errors", label: "错误日志", icon: Bug },
  { to: "/admin/system", label: "系统指标", icon: Wrench },
];

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

function handleAdminTokenChange(value: string) {
  saveAdminToken(value);
}

function handleCommand(command: string) {
  switch (command) {
    case "profile":
      router.push("/user/profile");
      break;
    case "switch-admin":
      if (session.adminToken) {
        router.push("/admin/dashboard");
      } else {
        router.push("/login/admin");
      }
      break;
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

<style scoped>
.sidebar-footer {
  padding: 20px;
  border-top: 1px solid var(--border-color);
}

.user-info-mini {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: var(--border-light);
  border-radius: var(--radius-lg);
}

.user-avatar-small {
  width: 40px;
  height: 40px;
  background: var(--primary-gradient);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 16px;
  flex-shrink: 0;
}

.user-details {
  flex: 1;
  min-width: 0;
}

.user-name-small {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-email-small {
  font-size: 12px;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.admin-token-input {
  width: 240px;
}

:deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  font-size: 14px;
}

:deep(.el-tag) {
  border-radius: var(--radius-md);
  font-weight: 500;
}
</style>
