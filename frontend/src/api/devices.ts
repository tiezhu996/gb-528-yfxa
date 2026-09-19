import { api, json, page } from './client'
import type { CreateDeviceInput, RiggingDevice, UpdateDeviceInput } from '../types/device'

export async function listDevices(): Promise<RiggingDevice[]> {
  return (await page<RiggingDevice>('/devices?page_size=200')).data
}

export async function createDevice(input: CreateDeviceInput): Promise<RiggingDevice> {
  return (await api<RiggingDevice>('/devices', json('POST', input))).data
}

export async function updateDevice(id: number, input: UpdateDeviceInput): Promise<RiggingDevice> {
  return (await api<RiggingDevice>(`/devices/${id}`, json('PUT', input))).data
}
