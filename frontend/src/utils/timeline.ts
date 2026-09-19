import type { TimelineEvent } from '../types/rehearsal'

export interface PositionedEvent extends TimelineEvent {
  leftPercent: number
  widthPercent: number
}

export function timelineExtent(events: TimelineEvent[]): number {
  return Math.max(1, ...events.map((event) => event.end_ms))
}

export function positionEvents(events: TimelineEvent[]): PositionedEvent[] {
  const extent = timelineExtent(events)
  return events.map((event) => ({
    ...event,
    leftPercent: (event.start_ms / extent) * 100,
    widthPercent: Math.max(1.5, ((event.end_ms - event.start_ms) / extent) * 100),
  }))
}

export function formatMillis(value: number): string {
  if (Math.abs(value) < 1000) return `${value} ms`
  return `${(value / 1000).toFixed(value % 1000 === 0 ? 0 : 1)} s`
}

export function formatTimestamp(value: string | null | undefined): string {
  if (!value) return 'Not recorded'
  return new Intl.DateTimeFormat('en-GB', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}
