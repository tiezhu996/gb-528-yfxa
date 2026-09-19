import { createRouter, createWebHistory } from 'vue-router'
import AppShell from '../components/common/AppShell.vue'
import LoginPage from '../pages/LoginPage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginPage, meta: { public: true } },
    {
      path: '/',
      component: AppShell,
      redirect: '/devices',
      children: [
        { path: 'devices', name: 'devices', component: () => import('../pages/DevicesPage.vue') },
        { path: 'cues', name: 'cues', component: () => import('../pages/CuesPage.vue') },
        { path: 'rules', name: 'rules', component: () => import('../pages/RulesPage.vue') },
        { path: 'rehearsals', name: 'rehearsals', component: () => import('../pages/RehearsalsPage.vue') },
        { path: 'audit', name: 'audit', component: () => import('../pages/AuditPage.vue'), meta: { roles: ['safety_reviewer', 'admin'] } },
      ],
    },
    { path: '/:pathMatch(.*)*', redirect: '/devices' },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('rigging_cue_token')
  const rawUser = localStorage.getItem('rigging_cue_user')
  if (to.meta.public) return token && rawUser ? '/devices' : true
  if (!token || !rawUser) return { path: '/login', query: { redirect: to.fullPath } }
  const roles = to.meta.roles as string[] | undefined
  if (roles) {
    try {
      const role = (JSON.parse(rawUser) as { role?: string }).role
      if (!role || !roles.includes(role)) return '/devices'
    } catch {
      localStorage.removeItem('rigging_cue_token')
      localStorage.removeItem('rigging_cue_user')
      return '/login'
    }
  }
  return true
})

export default router
