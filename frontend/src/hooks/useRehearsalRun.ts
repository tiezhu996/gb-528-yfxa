import { onBeforeUnmount, ref } from 'vue'
import { useRehearsalStore } from '../stores/rehearsals'

export function useRehearsalRun() {
  const store = useRehearsalStore()
  const polling = ref(false)
  let timer: number | null = null

  function stopPolling() {
    if (timer !== null) window.clearInterval(timer)
    timer = null
    polling.value = false
  }

  function poll(runId: number) {
    stopPolling()
    polling.value = true
    timer = window.setInterval(async () => {
      try {
        const run = await store.refresh(runId)
        if (!['evaluated', 'pending_review'].includes(run.run_status)) stopPolling()
      } catch {
        stopPolling()
      }
    }, 5000)
  }

  onBeforeUnmount(stopPolling)
  return { store, polling, poll, stopPolling }
}
