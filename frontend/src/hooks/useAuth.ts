import { computed } from 'vue'
import { useAuthStore } from '../stores/auth'

export function useAuth() {
  const auth = useAuthStore()
  const canProgram = computed(() => auth.hasRole('programmer', 'admin'))
  const canReview = computed(() => auth.hasRole('safety_reviewer', 'admin'))
  return { auth, canProgram, canReview }
}
