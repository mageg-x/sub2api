import { createRouter, createWebHistory } from 'vue-router'

function hasAdminAccess(): boolean {
  return Boolean(localStorage.getItem('sub2api_admin_token'))
}

function defaultAuthedRoute(): string {
  return hasAdminAccess() ? '/admin/dashboard' : '/user/dashboard'
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/auth/LoginView.vue')
    },
    {
      path: '/',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        { path: '', redirect: () => defaultAuthedRoute() },
        { path: '/admin/dashboard', name: 'admin-dashboard', component: () => import('@/views/admin/DashboardView.vue') },
        { path: '/admin/users', name: 'admin-users', component: () => import('@/views/admin/UsersView.vue') },
        { path: '/admin/api-keys', name: 'admin-api-keys', component: () => import('@/views/admin/APIKeysView.vue') },
        { path: '/admin/accounts', name: 'admin-accounts', component: () => import('@/views/admin/AccountsView.vue') },
        { path: '/admin/usage', name: 'admin-usage', component: () => import('@/views/admin/UsageView.vue') },
        { path: '/admin/prices', name: 'admin-prices', component: () => import('@/views/admin/PricesView.vue') },
        { path: '/admin/payments', name: 'admin-payments', component: () => import('@/views/admin/PaymentsView.vue') },
        { path: '/admin/announcements', name: 'admin-announcements', component: () => import('@/views/admin/AnnouncementsView.vue') },
        { path: '/admin/coupons', name: 'admin-coupons', component: () => import('@/views/admin/CouponsView.vue') },
        { path: '/admin/errors', name: 'admin-errors', component: () => import('@/views/admin/ErrorsView.vue') },
        { path: '/admin/system', name: 'admin-system', component: () => import('@/views/admin/SystemView.vue') },
        { path: '/user/dashboard', name: 'user-dashboard', component: () => import('@/views/user/DashboardView.vue') },
        { path: '/user/keys', name: 'user-keys', component: () => import('@/views/user/KeysView.vue') },
        { path: '/user/usage', name: 'user-usage', component: () => import('@/views/user/UsageView.vue') },
        { path: '/user/payment', name: 'user-payment', component: () => import('@/views/user/PaymentView.vue') },
        { path: '/user/profile', name: 'user-profile', component: () => import('@/views/user/ProfileView.vue') },
        { path: '/user/announcements', name: 'user-announcements', component: () => import('@/views/user/AnnouncementsView.vue') },
        { path: '/user/redeem', name: 'user-redeem', component: () => import('@/views/user/RedeemView.vue') },
        { path: '/user/access-guide', name: 'user-access-guide', component: () => import('@/views/user/AccessGuideView.vue') },
        { path: '/user/orders/:id', name: 'user-order-detail', component: () => import('@/views/user/OrderDetailView.vue') }
      ]
    }
  ]
})

router.beforeEach((to) => {
  const token = localStorage.getItem('sub2api_access_token')
  if (to.path !== '/login' && !token) {
    return '/login'
  }
  if (to.path.startsWith('/admin') && !hasAdminAccess()) {
    return '/user/dashboard'
  }
  if (to.path === '/login' && token) {
    return defaultAuthedRoute()
  }
  return true
})

export default router
