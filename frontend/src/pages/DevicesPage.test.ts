import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DevicesPage from './DevicesPage.vue'
import { tokenKey, userKey } from '../api/client'

const deviceAPI = vi.hoisted(() => ({
  listDevices: vi.fn(),
  fetchMaintenanceCheck: vi.fn(),
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
}))

vi.mock('../api/devices', () => deviceAPI)
vi.mock('element-plus', () => ({ ElMessage: { success: vi.fn() } }))

const ElTableStub = { name: 'ElTable', emits: ['current-change'], template: '<div><slot /></div>' }
const stubs = { 'el-table': ElTableStub, 'el-table-column': { template: '<span />' } }

const device = {
  id: 10,
  device_code: 'BATTEN-04',
  name: 'Test batten',
  device_type: 'motorized_batten',
  max_load_kg: 500,
  max_speed_ms: 0.4,
  travel_min_m: 4,
  travel_max_m: 16,
  safety_zone: 'overstage-c',
  device_status: 'available',
  version: 1,
  applicable_rules: [],
  created_at: '2026-08-22T00:00:00Z',
  updated_at: '2026-08-22T00:00:00Z',
}

function mountPage() {
  return mount(DevicesPage, {
    global: {
      plugins: [createPinia()],
      config: { warnHandler: () => undefined },
      stubs,
    },
  })
}

describe('DevicesPage actions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    localStorage.setItem(tokenKey, 'test-token')
    localStorage.setItem(userKey, JSON.stringify({ id: 1, username: 'programmer', display_name: 'Test Programmer', role: 'programmer' }))
    deviceAPI.listDevices.mockResolvedValue([])
    deviceAPI.fetchMaintenanceCheck.mockResolvedValue({ device_id: 10, device_code: 'BATTEN-04', blocked: false, locked_cues: [], enabled_rules: [] })
    deviceAPI.createDevice.mockImplementation(async (input) => ({
      ...input,
      id: 10,
      version: 1,
      applicable_rules: [],
      created_at: '2026-08-22T00:00:00Z',
      updated_at: '2026-08-22T00:00:00Z',
    }))
  })

  it('dispatches device creation from the explicit action button', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const createButton = wrapper.findAll('el-button').find((button) => button.text().includes('Create device'))
    expect(createButton).toBeDefined()
    await createButton!.trigger('click')
    await flushPromises()

    expect(deviceAPI.createDevice).toHaveBeenCalledTimes(1)
  })

  it('shows locked cue blockers when the maintenance freeze applies', async () => {
    deviceAPI.listDevices.mockResolvedValue([device])
    deviceAPI.fetchMaintenanceCheck.mockResolvedValue({
      device_id: 10,
      device_code: 'BATTEN-04',
      blocked: true,
      locked_cues: [{ id: 5, cue_code: 'Q-010', sequence_no: 10, version: 4 }],
      enabled_rules: [{ id: 2, rule_code: 'LOAD-ALL-01', rule_type: 'load_limit', severity: 'blocker', enabled: true, rule_version: 1 }],
    })
    const wrapper = mountPage()
    await flushPromises()

    wrapper.findComponent(ElTableStub).vm.$emit('current-change', device)
    await flushPromises()

    expect(deviceAPI.fetchMaintenanceCheck).toHaveBeenCalledWith(10)
    expect(wrapper.text()).toContain('MAINTENANCE FREEZE')
    expect(wrapper.text()).toContain('Q-010')
    expect(wrapper.text()).toContain('LOAD-ALL-01')
  })

  it('stays quiet when no locked cue references the selected device', async () => {
    deviceAPI.listDevices.mockResolvedValue([device])
    const wrapper = mountPage()
    await flushPromises()

    wrapper.findComponent(ElTableStub).vm.$emit('current-change', device)
    await flushPromises()

    expect(deviceAPI.fetchMaintenanceCheck).toHaveBeenCalledWith(10)
    expect(wrapper.text()).not.toContain('MAINTENANCE FREEZE')
  })
})
