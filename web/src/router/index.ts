import { createRouter, createWebHistory } from 'vue-router'

function hasAdminAccess(): boolean {
  return Boolean(localStorage.getItem('sub2api_admin_token'))
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('@/views/HomeView.vue')
    },
    {
      path: '/login',
      redirect: () => '/login/user'
    },
    {
      path: '/login/user',
      name: 'user-login',
      component: () => import('@/views/auth/UserLoginView.vue')
    },
    {
      path: '/login/admin',
      name: 'admin-login',
      component: () => import('@/views/auth/AdminLoginView.vue')
    },
    {
      path: '/user',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        { path: 'dashboard', name: 'user-dashboard', component: () => import('@/views/user/DashboardView.vue') },
        { path: 'keys', name: 'user-keys', component: () => import('@/views/user/KeysView.vue') },
        { path: 'usage', name: 'user-usage', component: () => import('@/views/user/UsageView.vue') },
        { path: 'payment', name: 'user-payment', component: () => import('@/views/user/PaymentView.vue') },
        { path: 'profile', name: 'user-profile', component: () => import('@/views/user/ProfileView.vue') },
        { path: 'announcements', name: 'user-announcements', component: () => import('@/views/user/AnnouncementsView.vue') },
        { path: 'redeem', name: 'user-redeem', component: () => import('@/views/user/RedeemView.vue') },
        { path: 'access-guide', name: 'user-access-guide', component: () => import('@/views/user/AccessGuideView.vue') },
        { path: 'orders/:id', name: 'user-order-detail', component: () => import('@/views/user/OrderDetailView.vue') }
      ]
    },
    {
      path: '/admin',
      component: () => import('@/layouts/AppShell.vue'),
      children: [
        { path: 'dashboard', name: 'admin-dashboard', component: () => import('@/views/admin/DashboardView.vue') },
        { path: 'users', name: 'admin-users', component: () => import('@/views/admin/UsersView.vue') },
        { path: 'api-keys', name: 'admin-api-keys', component: () => import('@/views/admin/APIKeysView.vue') },
        { path: 'accounts', name: 'admin-accounts', component: () => import('@/views/admin/AccountsView.vue') },
        { path: 'usage', name: 'admin-usage', component: () => import('@/views/admin/UsageView.vue') },
        { path: 'prices', name: 'admin-prices', component: () => import('@/views/admin/PricesView.vue') },
        { path: 'payments', name: 'admin-payments', component: () => import('@/views/admin/PaymentsView.vue') },
        { path: 'announcements', name: 'admin-announcements', component: () => import('@/views/admin/AnnouncementsView.vue') },
        { path: 'coupons', name: 'admin-coupons', component: () => import('@/views/admin/CouponsView.vue') },
        { path: 'errors', name: 'admin-errors', component: () => import('@/views/admin/ErrorsView.vue') },
        { path: 'system', name: 'admin-system', component: () => import('@/views/admin/SystemView.vue') }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/'
    }
  ]
})

router.beforeEach((to, from, next) => {
  if (to.path.startsWith('/login')) {
    next()
    return
  }
  
  if (to.path === '/') {
    next()
    return
  }
  
  const token = localStorage.getItem('sub2api_access_token')
  
  if (!token) {
    next('/login/user')
    return
  }
  
  if (to.path.startsWith('/admin') && !hasAdminAccess()) {
    next('/user/dashboard')
    return
  }
  
  next()
})

export default router
