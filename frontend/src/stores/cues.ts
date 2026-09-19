import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import * as api from '../api/cues'
import { errorMessage } from '../api/client'
import type { CreateCueInput, CueDefinition, UpdateCueInput } from '../types/cue'

export const useCueStore = defineStore('cues', () => {
  const items = ref<CueDefinition[]>([])
  const loading = ref(false)
  const error = ref('')
  const locked = computed(() => items.value.filter((item) => item.cue_status === 'locked'))

  async function load() {
    loading.value = true
    error.value = ''
    try {
      items.value = await api.listCues()
    } catch (cause) {
      error.value = errorMessage(cause)
      throw cause
    } finally {
      loading.value = false
    }
  }

  async function create(input: CreateCueInput) {
    const created = await api.createCue(input)
    upsert(created)
    return created
  }

  async function update(id: number, input: UpdateCueInput) {
    const updated = await api.updateCue(id, input)
    upsert(updated)
    return updated
  }

  async function transition(id: number, action: 'submit' | 'approve' | 'reject' | 'lock' | 'archive', version: number, reason: string) {
    const updated = await api.transitionCue(id, action, version, reason)
    upsert(updated)
    return updated
  }

  function upsert(cue: CueDefinition) {
    const existing = items.value.some((item) => item.id === cue.id)
    items.value = existing ? items.value.map((item) => (item.id === cue.id ? cue : item)) : [...items.value, cue]
    items.value.sort((a, b) => a.sequence_no - b.sequence_no || a.cue_code.localeCompare(b.cue_code))
  }

  return { items, locked, loading, error, load, create, update, transition }
})
