import type { ApiEnvelope, ApiErrorBody, PageEnvelope } from '../types/common'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    public readonly code: string,
    message: string,
    public readonly details: unknown,
    public readonly requestId: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export const tokenKey = 'rigging_cue_token'
export const userKey = 'rigging_cue_user'

async function call<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = localStorage.getItem(tokenKey)
  const headers = new Headers(options.headers)
  if (!headers.has('Content-Type') && options.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const response = await fetch(path, { ...options, headers })
  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    const body = payload as ApiErrorBody | null
    if (response.status === 401) {
      localStorage.removeItem(tokenKey)
      localStorage.removeItem(userKey)
      window.dispatchEvent(new Event('rigging-auth-expired'))
    }
    throw new ApiError(response.status, body?.error?.code ?? 'HTTP_ERROR', body?.error?.message ?? `Request failed with HTTP ${response.status}`, body?.error?.details, body?.request_id ?? response.headers.get('X-Request-ID') ?? '')
  }
  return payload as T
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<ApiEnvelope<T>> {
  return call<ApiEnvelope<T>>(`/api/v1${path}`, options)
}

export async function page<T>(path: string): Promise<PageEnvelope<T>> {
  return call<PageEnvelope<T>>(`/api/v1${path}`)
}

export const json = (method: string, body?: unknown): RequestInit => ({
  method,
  body: body === undefined ? undefined : JSON.stringify(body),
})

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    const suffix = error.requestId ? ` · request ${error.requestId}` : ''
    return `${error.code}: ${error.message}${suffix}`
  }
  return error instanceof Error ? error.message : 'The request could not be completed.'
}
