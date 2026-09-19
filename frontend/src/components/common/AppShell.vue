<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Archive, Cable, ClipboardCheck, LogOut, Route, ShieldCheck } from 'lucide-vue-next'
import SafetyBoundary from './SafetyBoundary.vue'
import { useAuthStore } from '../../stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const navItems = [
  { path: '/devices', label: 'Devices', icon: Cable },
  { path: '/cues', label: 'Cue desk', icon: Route },
  { path: '/rules', label: 'Interlocks', icon: ShieldCheck },
  { path: '/rehearsals', label: 'Rehearsals', icon: ClipboardCheck },
  { path: '/audit', label: 'Audit review', icon: Archive },
]
const nav = computed(() => navItems.filter((item) => item.path !== '/audit' || auth.hasRole('safety_reviewer', 'admin')))
const section = computed(() => nav.value.find((item) => route.path.startsWith(item.path))?.label ?? 'Workspace')

async function signOut() {
  auth.signOut()
  await router.replace('/login')
}

onMounted(() => window.addEventListener('rigging-auth-expired', signOut, { once: true }))
</script>

<template>
  <div class="app-shell">
    <aside class="app-sidebar">
      <div class="brand-block">
        <span class="brand-mark">RQ</span>
        <div><strong>Rigging Cue Desk</strong><span>Offline interlock review</span></div>
      </div>
      <nav aria-label="Primary workspace">
        <RouterLink v-for="item in nav" :key="item.path" :to="item.path" class="nav-link">
          <component :is="item.icon" :size="18" />{{ item.label }}
        </RouterLink>
      </nav>
      <div class="identity-block">
        <span>{{ auth.user?.display_name }}</span>
        <strong>{{ auth.user?.role.replaceAll('_', ' ') }}</strong>
        <el-button text :icon="LogOut" @click="signOut">Sign out</el-button>
      </div>
    </aside>
    <main class="app-main">
      <div class="context-strip"><span>WORKSPACE / {{ section.toUpperCase() }}</span><strong>NO DEVICE LINK</strong></div>
      <SafetyBoundary />
      <div class="page-wrap"><RouterView /></div>
    </main>
  </div>
</template>
