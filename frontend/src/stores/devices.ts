import { ref } from 'vue'
import { defineStore } from 'pinia'
import * as api from '../api/devices'
import { errorMessage } from '../api/client'
import type { CreateDeviceInput, DeviceFreezeReport, RiggingDevice, UpdateDeviceInput } from '../types/device'

export const useDeviceStore = defineStore('devices', () => {
  const items = ref<RiggingDevice[]>([])
  const freezeReports = ref<Record<number, DeviceFreezeReport>>({})
  const loading = ref(false)
  const error = ref('')

  async function load() {
    loading.value = true
    error.value = ''
    try {
      items.value = await api.listDevices()
    } catch (cause) {
      error.value = errorMessage(cause)
      throw cause
    } finally {
      loading.value = false
    }
  }

  async function loadFreezeReport(id: number) {
    const report = await api.fetchMaintenanceCheck(id)
    freezeReports.value = { ...freezeReports.value, [id]: report }
    return report
  }

  async function create(input: CreateDeviceInput) {
    const created = await api.createDevice(input)
    items.value = [...items.value, created].sort((a, b) => a.device_code.localeCompare(b.device_code))
    return created
  }

  async function update(id: number, input: UpdateDeviceInput) {
    const updated = await api.updateDevice(id, input)
    items.value = items.value.map((item) => (item.id === id ? updated : item))
    return updated
  }

  return { items, freezeReports, loading, error, load, loadFreezeReport, create, update }
})
