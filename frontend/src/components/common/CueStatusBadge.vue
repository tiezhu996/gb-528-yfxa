<script setup lang="ts">
import { computed } from 'vue'
import type { CueStatus } from '../../types/cue'

const props = defineProps<{ status: CueStatus }>()
const label = computed(() => ({
  draft: 'Draft',
  pending_review: 'Pending review',
  approved: 'Approved',
  locked: 'Locked',
  archived: 'Archived',
})[props.status])
</script>

<template>
  <span class="status-badge" :class="`status-${status}`"><span class="status-dot" />{{ label }}</span>
</template>
