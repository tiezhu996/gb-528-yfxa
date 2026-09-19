import { api, json, page } from './client'
import type { RehearsalRun } from '../types/rehearsal'

export async function listRuns(): Promise<RehearsalRun[]> {
  return (await page<RehearsalRun>('/rehearsals?page_size=100')).data
}

export async function getRun(id: number): Promise<RehearsalRun> {
  return (await api<RehearsalRun>(`/rehearsals/${id}`)).data
}

export async function runRehearsal(cueIds: number[]): Promise<RehearsalRun> {
  return (await api<RehearsalRun>('/rehearsals/run', json('POST', { cue_ids: cueIds }))).data
}

export async function submitRun(id: number, version: number, reason: string): Promise<RehearsalRun> {
  return (await api<RehearsalRun>(`/rehearsals/${id}/submit`, json('POST', { version, reason }))).data
}

export async function reviewRun(id: number, version: number, decision: 'approve' | 'reject', reason: string): Promise<RehearsalRun> {
  return (await api<RehearsalRun>(`/rehearsals/${id}/review`, json('POST', { version, decision, reason }))).data
}

export async function compareRuns(id: number, otherId: number): Promise<Record<string, unknown>> {
  return (await api<Record<string, unknown>>(`/rehearsals/${id}/compare?other_id=${otherId}`)).data
}
