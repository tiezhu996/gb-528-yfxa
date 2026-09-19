<script setup lang="ts">
import { computed } from 'vue'
import type { TimelineEvent } from '../../types/rehearsal'
import { formatMillis, positionEvents, timelineExtent } from '../../utils/timeline'

const props = defineProps<{ events: TimelineEvent[]; compact?: boolean }>()
const extent = computed(() => timelineExtent(props.events))
const groups = computed(() => {
  const map = new Map<string, ReturnType<typeof positionEvents>>()
  for (const event of positionEvents(props.events)) {
    const existing = map.get(event.device_code) ?? []
    existing.push(event)
    map.set(event.device_code, existing)
  }
  return [...map.entries()].map(([deviceCode, events]) => ({ deviceCode, deviceName: events[0]?.device_name ?? deviceCode, events }))
})
const marks = computed(() => [0, 0.25, 0.5, 0.75, 1].map((ratio) => ({ ratio, label: formatMillis(Math.round(extent.value * ratio)) })))
</script>

<template>
  <div class="timeline" :class="{ compact }" data-testid="timeline-track">
    <div v-if="groups.length === 0" class="empty-inline">Run a locked cue set to render its deterministic action timeline.</div>
    <template v-else>
      <div class="timeline-axis" aria-hidden="true">
        <span v-for="mark in marks" :key="mark.ratio" :style="{ left: `${mark.ratio * 100}%` }">{{ mark.label }}</span>
      </div>
      <div v-for="group in groups" :key="group.deviceCode" class="timeline-row">
        <div class="timeline-label">
          <strong>{{ group.deviceCode }}</strong>
          <span>{{ group.deviceName }}</span>
        </div>
        <div class="timeline-lane">
          <div
            v-for="event in group.events"
            :key="`${event.cue_id}-${event.device_id}-${event.start_ms}`"
            class="timeline-event"
            :style="{ left: `${event.leftPercent}%`, width: `${event.widthPercent}%` }"
            :title="`${event.cue_code}: ${formatMillis(event.start_ms)}–${formatMillis(event.end_ms)}, ${event.from_position_m}m → ${event.to_position_m}m`"
          >
            <span>{{ event.cue_code }}</span>
            <small>{{ event.from_position_m }}→{{ event.to_position_m }}m</small>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
