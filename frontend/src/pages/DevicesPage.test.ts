import { createPinia } from 'pinia'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DevicesPage from './DevicesPage.vue'
import { tokenKey, userKey } from '../api/client'

const deviceAPI = vi.hoisted(() => ({
  listDevices: vi.fn(),
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
}))

vi.mock('../api/devices', () => deviceAPI)
vi.mock('element-plus', () => ({ ElMessage: { success: vi.fn() } }))

describe('DevicesPage actions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    localStorage.setItem(tokenKey, 'test-token')
    localStorage.setItem(userKey, JSON.stringify({ id: 1, username: 'programmer', display_name: 'Test Programmer', role: 'programmer' }))
    deviceAPI.listDevices.mockResolvedValue([])
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
    const wrapper = mount(DevicesPage, {
      global: {
        plugins: [createPinia()],
        config: { warnHandler: () => undefined },
        stubs: {
          'el-table': { template: '<div><slot /></div>' },
          'el-table-column': { template: '<span />' },
        },
      },
    })
    await flushPromises()

    const createButton = wrapper.findAll('el-button').find((button) => button.text().includes('Create device'))
    expect(createButton).toBeDefined()
    await createButton!.trigger('click')
    await flushPromises()

    expect(deviceAPI.createDevice).toHaveBeenCalledTimes(1)
  })

  it('shows the pre-maintenance freeze reason with the locked cue code', async () => {
    deviceAPI.listDevices.mockResolvedValue([
      {
        id: 4,
        device_code: 'HOIST-FRZ-09',
        name: 'Freeze test hoist',
        device_type: 'point_hoist',
        max_load_kg: 500,
        max_speed_ms: 0.5,
        travel_min_m: 2,
        travel_max_m: 12,
        safety_zone: 'zone-x',
        device_status: 'available',
        version: 1,
        applicable_rules: [],
        maintenance_freeze: {
          target_status: 'inspection_hold',
          blocked: true,
          locked_cues: [{ id: 40, cue_code: 'Q-FRZ-090', sequence_no: 90, version: 3 }],
          enabled_interlock_rules: [],
          required_actions: ['archive_locked_cue', 'revise_locked_cue_to_new_version'],
        },
        created_at: '2026-08-22T00:00:00Z',
        updated_at: '2026-08-22T00:00:00Z',
      },
    ])
    const wrapper = mount(DevicesPage, {
      global: {
        plugins: [createPinia()],
        config: { warnHandler: () => undefined },
        stubs: {
          'el-table': {
            props: ['data'],
            template: '<div><slot /><button v-for="row in data" :key="row.id" class="select-row" @click="$emit(\'current-change\', row)">select</button></div>',
          },
          'el-table-column': { template: '<span />' },
        },
      },
    })
    await flushPromises()
    await wrapper.find('.select-row').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Q-FRZ-090')
    expect(wrapper.text()).toContain('Referencing locked cues (1)')
    expect(wrapper.find('.rule-reference .blocker').exists()).toBe(true)
  })
})
