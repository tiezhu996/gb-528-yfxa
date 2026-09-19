import { api, json, page } from './client'
import type { CreateCueInput, CueDefinition, UpdateCueInput } from '../types/cue'

export async function listCues(): Promise<CueDefinition[]> {
  return (await page<CueDefinition>('/cues?page_size=200')).data
}

export async function createCue(input: CreateCueInput): Promise<CueDefinition> {
  return (await api<CueDefinition>('/cues', json('POST', input))).data
}

export async function updateCue(id: number, input: UpdateCueInput): Promise<CueDefinition> {
  return (await api<CueDefinition>(`/cues/${id}`, json('PUT', input))).data
}

export async function transitionCue(id: number, action: 'submit' | 'approve' | 'reject' | 'lock' | 'archive', version: number, reason: string): Promise<CueDefinition> {
  return (await api<CueDefinition>(`/cues/${id}/${action}`, json('POST', { version, reason }))).data
}
