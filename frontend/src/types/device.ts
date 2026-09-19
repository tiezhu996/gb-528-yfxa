export type DeviceStatus = 'available' | 'inspection_hold' | 'retired'
export type DeviceType = 'motorized_batten' | 'motorized_bridge' | 'scenic_carrier' | 'point_hoist' | 'manual_counterweight'

export interface RuleReference {
  id: number
  rule_code: string
  rule_type: string
  severity: 'warning' | 'blocker'
  enabled: boolean
  rule_version: number
}

export interface RiggingDevice {
  id: number
  device_code: string
  name: string
  device_type: DeviceType
  max_load_kg: number
  max_speed_ms: number
  travel_min_m: number
  travel_max_m: number
  safety_zone: string
  device_status: DeviceStatus
  version: number
  applicable_rules: RuleReference[]
  created_at: string
  updated_at: string
}

export type CreateDeviceInput = Omit<RiggingDevice, 'id' | 'version' | 'applicable_rules' | 'created_at' | 'updated_at'>
export type UpdateDeviceInput = Omit<CreateDeviceInput, 'device_code'> & { version: number }
