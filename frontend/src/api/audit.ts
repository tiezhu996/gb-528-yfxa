import { page } from './client'
import type { AuditEvent } from '../types/audit'

export async function listAuditEvents(search = ''): Promise<AuditEvent[]> {
  const query = search ? `&search=${encodeURIComponent(search)}` : ''
  return (await page<AuditEvent>(`/audit-events?page_size=200${query}`)).data
}
