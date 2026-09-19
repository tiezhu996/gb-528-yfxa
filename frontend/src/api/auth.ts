import { api, json } from './client'
import type { LoginResponse } from '../types/auth'

export async function login(username: string, password: string): Promise<LoginResponse> {
  return (await api<LoginResponse>('/auth/login', json('POST', { username, password }))).data
}
