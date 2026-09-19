import { ref } from 'vue'
import { defineStore } from 'pinia'
import * as api from '../api/rules'
import { errorMessage } from '../api/client'
import type { InterlockRule, TestRuleResponse } from '../types/interlock'

export const useRuleStore = defineStore('rules', () => {
  const items = ref<InterlockRule[]>([])
  const tests = ref<Record<number, TestRuleResponse>>({})
  const loading = ref(false)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      items.value = await api.listRules()
    } catch (cause) {
      error.value = errorMessage(cause)
      throw cause
    } finally {
      loading.value = false
    }
  }

  async function create(input: api.CreateRuleInput) {
    const created = await api.createRule(input)
    upsert(created)
    return created
  }

  async function update(id: number, input: api.UpdateRuleInput) {
    const updated = await api.updateRule(id, input)
    upsert(updated)
    return updated
  }

  async function toggle(rule: InterlockRule, enabled: boolean) {
    const updated = await api.toggleRule(rule.id, enabled, rule.rule_version)
    upsert(updated)
    return updated
  }

  async function test(rule: InterlockRule, actual: number) {
    const result = await api.testRule(rule.id, actual)
    tests.value = { ...tests.value, [rule.id]: result }
    return result
  }

  function upsert(rule: InterlockRule) {
    const existing = items.value.some((item) => item.id === rule.id)
    items.value = existing ? items.value.map((item) => (item.id === rule.id ? rule : item)) : [...items.value, rule]
    items.value.sort((a, b) => a.rule_code.localeCompare(b.rule_code))
  }

  return { items, tests, loading, error, load, create, update, toggle, test }
})
