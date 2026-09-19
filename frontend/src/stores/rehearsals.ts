import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as api from '../api/rehearsals'
import { errorMessage } from '../api/client'
import type { RehearsalRun } from '../types/rehearsal'

export const useRehearsalStore = defineStore('rehearsals', () => {
  const items = ref<RehearsalRun[]>([])
  const selectedId = ref<number | null>(null)
  const loading = ref(false)
  const running = ref(false)
  const error = ref('')
  const selected = computed(() => items.value.find((item) => item.id === selectedId.value) ?? null)

  async function load() {
    loading.value = true
    error.value = ''
    try {
      items.value = await api.listRuns()
      if (!selectedId.value && items.value[0]) selectedId.value = items.value[0].id
    } catch (cause) {
      error.value = errorMessage(cause)
      throw cause
    } finally {
      loading.value = false
    }
  }

  async function refresh(id: number) {
    const item = await api.getRun(id)
    upsert(item)
    return item
  }

  async function run(cueIds: number[]) {
    running.value = true
    error.value = ''
    try {
      const created = await api.runRehearsal(cueIds)
      upsert(created)
      selectedId.value = created.id
      return created
    } catch (cause) {
      error.value = errorMessage(cause)
      throw cause
    } finally {
      running.value = false
    }
  }

  async function submit(item: RehearsalRun, reason: string) {
    const updated = await api.submitRun(item.id, item.version, reason)
    upsert(updated)
    return updated
  }

  async function review(item: RehearsalRun, decision: 'approve' | 'reject', reason: string) {
    const updated = await api.reviewRun(item.id, item.version, decision, reason)
    upsert(updated)
    return updated
  }

  function upsert(run: RehearsalRun) {
    const exists = items.value.some((item) => item.id === run.id)
    items.value = exists ? items.value.map((item) => (item.id === run.id ? run : item)) : [run, ...items.value]
    items.value.sort((a, b) => b.id - a.id)
  }

  return { items, selectedId, selected, loading, running, error, load, refresh, run, submit, review }
})
