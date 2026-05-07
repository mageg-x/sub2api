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
            <p class="sidebar-subtitle">{{ t("userShell.userPanel") }}</p>
          </div>
        </div>
      </div>

      <nav class="sidebar-nav">
        <div class="nav-section">
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
            <div class="user-name-small">{{ session.user?.name || t("common.defaultUser") }}</div>
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
          <el-button class="header-ghost-button" link @click="announcementDialogVisible = true">
            <Bell :size="16" />
            {{ t("user.announcement") }}
          </el-button>
          <el-tag type="success" effect="dark" size="large">{{ t("common.user") }}</el-tag>
          <el-tag type="warning" effect="dark" size="large">{{ t("userShell.balanceLabel") }} {{ formatCurrency(session.user?.balance || 0) }} {{ t("common.currency") }}</el-tag>
          <el-dropdown @command="handleCommand">
            <div class="user-menu">
              <div class="user-avatar">{{ userInitials }}</div>
              <div class="user-info">
                <span class="user-name">{{ session.user?.name || t("common.defaultUser") }}</span>
                <span class="user-role">{{ session.user?.role || "user" }}</span>
              </div>
              <ChevronDown :size="16" />
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <User :size="16" />
                  {{ t("user.personalProfile") }}
                </el-dropdown-item>
                <el-dropdown-item command="switch-admin">
                  <ShieldCheck :size="16" />
                  {{ t("user.switchToAdmin") }}
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

  <el-dialog v-model="announcementDialogVisible" :title="t('user.announcement')" width="760px" class="announcement-dialog">
    <div class="dialog-meta">
      <span class="dialog-count">{{ announcements.length }} {{ t("announcements.announcementsCount") }}</span>
    </div>
    <el-empty v-if="announcementLoading" :description="t('common.loading')" />
    <el-empty v-else-if="announcements.length === 0" :description="t('common.noAnnouncements')" />
    <el-timeline v-else class="announcement-timeline">
      <el-timeline-item v-for="item in announcements" :key="item.id" :timestamp="formatTime(item.published_at_ms)" placement="top">
        <div class="announcement-card">
          <div class="announcement-header">
            <h4 class="announcement-title">{{ item.title }}</h4>
            <el-tag :type="item.status === 'active' ? 'success' : 'info'" size="small">
              {{ item.status === "active" ? t("common.active") : t("common.ended") }}
            </el-tag>
          </div>
          <p class="announcement-content">{{ item.content }}</p>
        </div>
      </el-timeline-item>
    </el-timeline>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { RouterView, useRoute, useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import { Bell, BookOpenText, ChevronDown, Gift, KeyRound, LayoutDashboard, LogOut, ShieldCheck, Sparkles, User, UserCog, WalletCards, Bolt } from "lucide-vue-next";
import { ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElEmpty, ElTag, ElTimeline, ElTimelineItem } from "element-plus";
import logoUrl from "@/assets/logo.svg";
import { adminAPI } from "@/api/admin";
import { me, logout } from "@/api/auth";
import LanguageSwitcher from "@/components/LanguageSwitcher.vue";
import type { Announcement } from "@/api/types";
import { clearAuth, session } from "@/store/session";
import { formatCurrency, formatTime } from "@/utils";

const route = useRoute();
const router = useRouter();
const { t } = useI18n();

const userLinks = computed(() => [
  { to: "/user/dashboard", label: t("userShell.home"), icon: LayoutDashboard },
  { to: "/user/models", label: t("user.modelSquare"), icon: Sparkles },
  { to: "/user/keys", label: t("user.tokenManagement"), icon: KeyRound },
  { to: "/user/usage", label: t("user.dataDashboard"), icon: Bolt },
  { to: "/user/payment", label: t("user.payment"), icon: WalletCards },
  { to: "/user/redeem", label: t("user.redeem"), icon: Gift },
  { to: "/user/access-guide", label: t("user.accessGuide"), icon: BookOpenText },
  { to: "/user/profile", label: t("user.personalProfile"), icon: UserCog },
]);

type RouteMetaItem = {
  title: string;
  subtitle: string;
};

type UserRoutePath =
  | "/user/dashboard"
  | "/user/keys"
  | "/user/usage"
  | "/user/payment"
  | "/user/models"
  | "/user/profile"
  | "/user/redeem"
  | "/user/access-guide"
  | "/user/announcements";

const routeMeta = computed<Record<UserRoutePath, RouteMetaItem>>(() => ({
  "/user/dashboard": { title: t("userShell.home"), subtitle: t("userShell.homeSubtitle") },
  "/user/keys": { title: t("user.tokenManagement"), subtitle: t("userShell.keysSubtitle") },
  "/user/usage": { title: t("user.dataDashboard"), subtitle: t("userShell.usageSubtitle") },
  "/user/payment": { title: t("user.payment"), subtitle: t("userShell.paymentSubtitle") },
  "/user/models": { title: t("user.modelSquare"), subtitle: t("userShell.modelsSubtitle") },
  "/user/profile": { title: t("user.personalProfile"), subtitle: t("userShell.profileSubtitle") },
  "/user/redeem": { title: t("user.redeem"), subtitle: t("userShell.redeemSubtitle") },
  "/user/access-guide": { title: t("user.accessGuide"), subtitle: t("userShell.accessGuideSubtitle") },
  "/user/announcements": { title: t("user.announcement"), subtitle: t("userShell.announcementsSubtitle") },
}));

function getRouteMeta(path: string): RouteMetaItem | undefined {
  if (path in routeMeta.value) {
    return routeMeta.value[path as UserRoutePath];
  }
  return undefined;
}

const pageTitle = computed(() => {
  return getRouteMeta(route.path)?.title || "sub2api";
});

const pageSubtitle = computed(() => {
  return getRouteMeta(route.path)?.subtitle || "";
});

const userInitials = computed(() => {
  const name = session.user?.name || session.user?.email || "U";
  return name.charAt(0).toUpperCase();
});

const announcementDialogVisible = ref(false);
const announcementLoading = ref(false);
const announcements = ref<Announcement[]>([]);

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

async function loadAnnouncements() {
  if (announcementLoading.value || announcements.value.length) return;
  announcementLoading.value = true;
  try {
    announcements.value = await adminAPI.announcements();
  } finally {
    announcementLoading.value = false;
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
  void loadAnnouncements();
});
</script>

<style scoped>
.header-ghost-button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-right: 4px;
  color: var(--text-secondary);
}

.header-ghost-button:hover {
  color: var(--primary-color);
}

.dialog-meta {
  margin-bottom: 16px;
}

.dialog-count {
  color: var(--text-muted);
  font-size: 13px;
}

.announcement-timeline {
  padding-top: 8px;
}

.announcement-card {
  background: var(--bg-subtle);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  padding: 16px 18px;
}

.announcement-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.announcement-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.announcement-content {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.7;
  white-space: pre-wrap;
}
</style>
