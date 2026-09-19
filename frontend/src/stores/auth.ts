import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { login as loginRequest } from '../api/auth'
import { tokenKey, userKey } from '../api/client'
import type { AuthUser, UserRole } from '../types/auth'

function restoreUser(): AuthUser | null {
  const raw = localStorage.getItem(userKey)
  if (!raw) return null
  try {
    return JSON.parse(raw) as AuthUser
  } catch {
    localStorage.removeItem(userKey)
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(tokenKey) ?? '')
  const user = ref<AuthUser | null>(restoreUser())
  const loading = ref(false)
  const authenticated = computed(() => Boolean(token.value && user.value))

  async function signIn(username: string, password: string) {
    loading.value = true
    try {
      const response = await loginRequest(username, password)
      token.value = response.access_token
      user.value = response.user
      localStorage.setItem(tokenKey, response.access_token)
      localStorage.setItem(userKey, JSON.stringify(response.user))
    } finally {
      loading.value = false
    }
  }

  function signOut() {
    token.value = ''
    user.value = null
    localStorage.removeItem(tokenKey)
    localStorage.removeItem(userKey)
  }

  function hasRole(...roles: UserRole[]) {
    return user.value ? roles.includes(user.value.role) : false
  }

  return { token, user, loading, authenticated, signIn, signOut, hasRole }
})
