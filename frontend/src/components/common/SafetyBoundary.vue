<script setup lang="ts">
import { ShieldAlert } from 'lucide-vue-next'
</script>

<template>
  <div class="safety-boundary" role="note">
    <ShieldAlert :size="18" aria-hidden="true" />
    <div>
      <strong>Offline rehearsal evidence</strong>
      <span>No hoist, console, PLC, or stage machinery connection. Nothing here is an executable motion command or operational clearance.</span>
    </div>
  </div>
</template>
