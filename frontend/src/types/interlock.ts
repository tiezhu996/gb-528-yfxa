export type InterlockResult = 'pass' | 'warning' | 'blocker' | 'invalid'
export type RuleType = 'load_limit' | 'speed_limit' | 'travel_limit' | 'zone_exclusion' | 'dependency_guard'

export interface RuleThreshold {
  max_load_kg?: number
  max_speed_ms?: number
  min_position_m?: number
  max_position_m?: number
  minimum_gap_ms?: number
  use_device_limits?: boolean
}

export interface InterlockRule {
  id: number
  rule_code: string
  rule_type: RuleType
  device_ids: number[]
  threshold: RuleThreshold
  severity: 'warning' | 'blocker'
  enabled: boolean
  rule_version: number
  explanation: string
  created_at: string
  updated_at: string
}

export interface RuleEvidence {
  rule_code: string
  rule_type: string
  result: InterlockResult
  severity: string
  cue_codes: string[] | null
  device_codes: string[] | null
  window_start_ms: number
  window_end_ms: number
  actual_value: number
  threshold_value: number
  unit: string
  message: string
}

export interface TestRuleResponse {
  rule_code: string
  rule_type: string
  result: InterlockResult
  actual_value: number
  threshold_value: number
  unit: string
  explanation: string
  boundary: string
}
