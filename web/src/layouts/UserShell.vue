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
          <el-button class="header-ghost-button" link @click="announcementDialogVisible = true">
            <Bell :size="16" />
            公告
          </el-button>
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

  <el-dialog v-model="announcementDialogVisible" title="公告" width="760px" class="announcement-dialog">
    <div class="dialog-meta">
      <span class="dialog-count">{{ announcements.length }} 条公告</span>
    </div>
    <el-empty v-if="announcementLoading" description="加载中" />
    <el-empty v-else-if="announcements.length === 0" description="暂无公告" />
    <el-timeline v-else class="announcement-timeline">
      <el-timeline-item v-for="item in announcements" :key="item.id" :timestamp="formatTime(item.published_at_ms)" placement="top">
        <div class="announcement-card">
          <div class="announcement-header">
            <h4 class="announcement-title">{{ item.title }}</h4>
            <el-tag :type="item.status === 'active' ? 'success' : 'info'" size="small">
              {{ item.status === "active" ? "进行中" : "已结束" }}
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
import { Bell, BookOpenText, ChevronDown, Gift, KeyRound, LayoutDashboard, LogOut, ShieldCheck, Sparkles, User, UserCog, WalletCards, Bolt } from "lucide-vue-next";
import { ElDialog, ElDropdown, ElDropdownItem, ElDropdownMenu, ElEmpty, ElTag, ElTimeline, ElTimelineItem } from "element-plus";
import logoUrl from "@/assets/logo.svg";
import { adminAPI } from "@/api/admin";
import { me, logout } from "@/api/auth";
import type { Announcement } from "@/api/types";
import { clearAuth, session } from "@/store/session";
import { formatCurrency, formatTime } from "@/utils";

const route = useRoute();
const router = useRouter();

const userLinks = [
  { to: "/user/dashboard", label: "首页", icon: LayoutDashboard },
  { to: "/user/models", label: "模型广场", icon: Sparkles },
  { to: "/user/keys", label: "令牌管理", icon: KeyRound },
  { to: "/user/usage", label: "数据看板", icon: Bolt },
  { to: "/user/payment", label: "充值", icon: WalletCards },
  { to: "/user/redeem", label: "兑换码", icon: Gift },
  { to: "/user/access-guide", label: "接入指南", icon: BookOpenText },
  { to: "/user/profile", label: "个人资料", icon: UserCog },
];

const routeMeta: Record<string, { title: string; subtitle: string }> = {
  "/user/dashboard": { title: "首页", subtitle: "您的账户概览" },
  "/user/keys": { title: "令牌管理", subtitle: "管理您的 API Keys" },
  "/user/usage": { title: "数据看板", subtitle: "查看使用记录" },
  "/user/payment": { title: "充值", subtitle: "充值余额" },
  "/user/models": { title: "模型广场", subtitle: "查看可用模型和价格" },
  "/user/profile": { title: "个人资料", subtitle: "个人资料设置" },
  "/user/redeem": { title: "兑换码", subtitle: "兑换码兑换" },
  "/user/access-guide": { title: "接入指南", subtitle: "API 接入指南" },
  "/user/announcements": { title: "公告", subtitle: "查看公告" },
};

const pageTitle = computed(() => {
  return routeMeta[route.path]?.title || "sub2api";
});

const pageSubtitle = computed(() => {
  return routeMeta[route.path]?.subtitle || "";
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
