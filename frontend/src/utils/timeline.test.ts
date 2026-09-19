import { describe, expect, it } from 'vitest'
import { formatMillis, positionEvents, timelineExtent } from './timeline'
import type { TimelineEvent } from '../types/rehearsal'

const event = (start_ms: number, end_ms: number): TimelineEvent => ({ cue_id: 1, cue_code: 'Q-1', cue_sequence: 1, cue_version: 4, device_id: 1, device_code: 'D-1', device_name: 'Device', safety_zone: 'zone-a', start_ms, end_ms, from_position_m: 10, to_position_m: 8, load_kg: 100, speed_ms: 0.2 })

describe('timeline helpers', () => {
  it('uses the latest end as the stable extent', () => {
    expect(timelineExtent([event(0, 1000), event(2000, 5000)])).toBe(5000)
  })

  it('positions events without changing source timing', () => {
    const positioned = positionEvents([event(1000, 3000), event(3000, 5000)])
    expect(positioned[0].leftPercent).toBe(20)
    expect(positioned[0].widthPercent).toBe(40)
    expect(positioned[1].leftPercent).toBe(60)
  })

  it('formats milliseconds and seconds', () => {
    expect(formatMillis(850)).toBe('850 ms')
    expect(formatMillis(2500)).toBe('2.5 s')
    expect(formatMillis(3000)).toBe('3 s')
  })
})
