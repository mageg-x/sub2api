<template>
  <div class="page-shell">
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="sidebar-brand">
          <div class="sidebar-logo">
            <img :src="logoUrl" alt="sub2api" />
          </div>
          <div>
            <h1 class="sidebar-title">sub2api</h1>
            <p class="sidebar-subtitle">{{ t("adminShell.adminConsole") }}</p>
          </div>
        </div>
      </div>

      <nav class="sidebar-nav">
        <div class="nav-section">
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
            <div class="user-name-small">{{ session.user?.name || t("common.defaultAdmin") }}</div>
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
          <LanguageSwitcher />
          <el-tag type="danger" effect="dark">{{ t("common.admin") }}</el-tag>

          <el-dropdown @command="handleCommand">
            <div class="user-menu">
              <div class="user-avatar">{{ userInitials }}</div>
              <div class="user-info">
                <span class="user-name">{{ session.user?.name || t("common.defaultAdmin") }}</span>
                <span class="user-role">{{ session.user?.role || "admin" }}</span>
              </div>
              <ChevronDown :size="16" />
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="switch-user">
                  <User :size="16" />
                  {{ t("admin.switchToUser") }}
                </el-dropdown-item>
                <el-dropdown-item divided command="logout" style="color: var(--danger-color)">
                  <LogOut :size="16" />
                  {{ t("auth.logout") }}
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
import { Banknote, Bug, ChevronDown, Gauge, LayoutDashboard, LogOut, Megaphone, ReceiptText, ShieldUser, Ticket, Users, User } from "lucide-vue-next";
import { ElDropdown, ElDropdownItem, ElDropdownMenu, ElTag } from "element-plus";
import logoUrl from "@/assets/logo.svg";
import { me, logout } from "@/api/auth";
import LanguageSwitcher from "@/components/LanguageSwitcher.vue";
import { clearAuth, session } from "@/store/session";
import { useI18n } from "vue-i18n";

const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const adminLinks = computed(() => [
  { to: "/admin/dashboard", label: t("admin.overview"), icon: LayoutDashboard },
  { to: "/admin/users", label: t("admin.users"), icon: Users },
  { to: "/admin/accounts", label: t("admin.upstreamAccounts"), icon: ShieldUser },
  { to: "/admin/prices", label: t("admin.priceConfig"), icon: Banknote },
  { to: "/admin/payments", label: t("admin.paymentOrders"), icon: ReceiptText },
  { to: "/admin/announcements", label: t("admin.announcementManagement"), icon: Megaphone },
  { to: "/admin/coupons", label: t("admin.couponCodes"), icon: Ticket },
  { to: "/admin/errors", label: t("admin.errorLogs"), icon: Bug },
  { to: "/admin/system", label: t("admin.systemMetrics"), icon: Gauge },
]);

const pageTitle = computed(() => {
  const match = adminLinks.value.find((item) => item.to === route.path);
  return match?.label || "sub2api";
});

const pageSubtitle = computed(() => {
  const descriptions: Record<string, string> = {
    "/admin/dashboard": t("adminShell.dashboardSubtitle"),
    "/admin/users": t("adminShell.usersSubtitle"),
    "/admin/accounts": t("adminShell.accountsSubtitle"),
    "/admin/prices": t("adminShell.pricesSubtitle"),
    "/admin/payments": t("adminShell.paymentsSubtitle"),
    "/admin/announcements": t("adminShell.announcementsSubtitle"),
    "/admin/coupons": t("adminShell.couponsSubtitle"),
    "/admin/errors": t("adminShell.errorsSubtitle"),
    "/admin/system": t("adminShell.systemSubtitle"),
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
