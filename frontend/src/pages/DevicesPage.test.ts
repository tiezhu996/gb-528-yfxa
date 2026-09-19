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
})
