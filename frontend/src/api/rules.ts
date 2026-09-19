import { api, json, page } from './client'
import type { InterlockRule, RuleThreshold, RuleType, TestRuleResponse } from '../types/interlock'

export interface CreateRuleInput {
  rule_code: string
  rule_type: RuleType
  device_ids: number[]
  threshold: RuleThreshold
  severity: 'warning' | 'blocker'
  enabled: boolean
  explanation: string
}

export type UpdateRuleInput = Omit<CreateRuleInput, 'rule_code' | 'rule_type'> & { rule_version: number }

export async function listRules(): Promise<InterlockRule[]> {
  return (await page<InterlockRule>('/rules?page_size=200')).data
}

export async function createRule(input: CreateRuleInput): Promise<InterlockRule> {
  return (await api<InterlockRule>('/rules', json('POST', input))).data
}

export async function updateRule(id: number, input: UpdateRuleInput): Promise<InterlockRule> {
  return (await api<InterlockRule>(`/rules/${id}`, json('PUT', input))).data
}

export async function toggleRule(id: number, enabled: boolean, ruleVersion: number): Promise<InterlockRule> {
  return (await api<InterlockRule>(`/rules/${id}/toggle`, json('POST', { enabled, rule_version: ruleVersion }))).data
}

export async function testRule(id: number, actualValue: number): Promise<TestRuleResponse> {
  return (await api<TestRuleResponse>(`/rules/${id}/test`, json('POST', { actual_value: actualValue }))).data
}
