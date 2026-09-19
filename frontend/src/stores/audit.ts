import { ref } from 'vue'
import { defineStore } from 'pinia'
import { listAuditEvents } from '../api/audit'
import { errorMessage } from '../api/client'
import type { AuditEvent } from '../types/audit'

export const useAuditStore = defineStore('audit', () => {
  const items = ref<AuditEvent[]>([])
  const loading = ref(false)
  const error = ref('')

  async function load(search = '') {
    loading.value = true
    error.value = ''
    try {
      items.value = await listAuditEvents(search)
    } catch (cause) {
      error.value = errorMessage(cause)
      throw cause
    } finally {
      loading.value = false
    }
  }

  return { items, loading, error, load }
})
