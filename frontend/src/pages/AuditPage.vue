<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { CheckCircle2, GitCompareArrows, Search, XCircle } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import PageHeader from '../components/common/PageHeader.vue'
import CueStatusBadge from '../components/common/CueStatusBadge.vue'
import { compareRuns } from '../api/rehearsals'
import { errorMessage } from '../api/client'
import { useAuditStore } from '../stores/audit'
import { useCueStore } from '../stores/cues'
import { useRehearsalStore } from '../stores/rehearsals'
import type { RehearsalRun } from '../types/rehearsal'
import { formatTimestamp } from '../utils/timeline'

const audit = useAuditStore()
const cues = useCueStore()
const runs = useRehearsalStore()
const search = ref('')
const reason = ref('Safety reviewer inspected the immutable evidence and offline boundary.')
const localError = ref('')
const compareLeft = ref<number | null>(null)
const compareRight = ref<number | null>(null)
const comparison = ref<Record<string, unknown> | null>(null)
const pending = computed(() => runs.items.filter((item) => item.run_status === 'pending_review'))

async function loadAudit() {
  localError.value = ''
  try {
    await audit.load(search.value)
  } catch (cause) {
    localError.value = errorMessage(cause)
  }
}

async function review(item: RehearsalRun, decision: 'approve' | 'reject') {
  localError.value = ''
  try {
    const updated = await runs.review(item, decision, reason.value)
    await audit.load(search.value)
    ElMessage.success(`Run #${item.id} ${updated.run_status.replaceAll('_', ' ')}`)
  } catch (cause) {
    localError.value = errorMessage(cause)
  }
}

async function compare() {
  if (!compareLeft.value || !compareRight.value) return
  localError.value = ''
  try {
    comparison.value = await compareRuns(compareLeft.value, compareRight.value)
  } catch (cause) {
    localError.value = errorMessage(cause)
  }
}

function formatSummary(value: string): string {
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return value
  }
}

onMounted(async () => {
  await Promise.all([audit.load(), cues.load(), runs.load()]).catch((cause) => { localError.value = errorMessage(cause) })
  compareLeft.value = runs.items[0]?.id ?? null
  compareRight.value = runs.items[1]?.id ?? null
})
</script>

<template>
  <PageHeader eyebrow="HUMAN REVIEW / APPEND-ONLY TRACE" title="Audit review" description="Compare immutable run versions, act on pending evidence, and trace every changed limit, cue, rule, and review decision by request ID." />
  <el-alert v-if="audit.error || cues.error || runs.error || localError" :title="localError || audit.error || cues.error || runs.error" type="error" :closable="false" show-icon />
  <section class="cue-state-strip">
    <div v-for="cue in cues.items" :key="cue.id"><span>{{ cue.cue_code }} · v{{ cue.version }}</span><CueStatusBadge :status="cue.cue_status" /></div>
  </section>
  <div class="audit-tools">
    <div class="audit-search"><Search :size="18" /><el-input v-model="search" placeholder="Actor, action, or request ID" clearable @keyup.enter="loadAudit" /><el-button @click="loadAudit">Filter</el-button></div>
    <div class="compare-tools"><GitCompareArrows :size="18" /><el-select v-model="compareLeft" placeholder="Left run"><el-option v-for="item in runs.items" :key="item.id" :value="item.id" :label="`#${item.id} · ${item.cue_set_version}`" /></el-select><el-select v-model="compareRight" placeholder="Right run"><el-option v-for="item in runs.items" :key="item.id" :value="item.id" :label="`#${item.id} · ${item.cue_set_version}`" /></el-select><el-button :disabled="!compareLeft || !compareRight || compareLeft === compareRight" @click="compare">Compare</el-button></div>
  </div>
  <section v-if="comparison" class="comparison-band"><p class="eyebrow">VERSION COMPARISON</p><pre>{{ JSON.stringify(comparison, null, 2) }}</pre></section>
  <section class="data-section pending-review">
    <div class="section-heading"><div><p class="eyebrow">REVIEW QUEUE</p><h2>{{ pending.length }} pending runs</h2></div><span>Reviewer-only decision</span></div>
    <el-input v-model="reason" type="textarea" :rows="2" maxlength="500" show-word-limit />
    <div v-if="pending.length === 0" class="empty-inline">No rehearsal evidence currently awaits review.</div>
    <div v-for="item in pending" :key="item.id" class="pending-row"><span><strong>Run #{{ item.id }}</strong><small>{{ item.cue_set_version }} · {{ item.highest_severity }}</small></span><el-button type="success" :icon="CheckCircle2" @click="review(item, 'approve')">Approve evidence</el-button><el-button :icon="XCircle" @click="review(item, 'reject')">Reject</el-button></div>
  </section>
  <section class="data-section audit-events">
    <div class="section-heading"><div><p class="eyebrow">EVENT LEDGER</p><h2>{{ audit.items.length }} audit events</h2></div><span>Append-only application history</span></div>
    <el-table :data="audit.items" :loading="audit.loading" stripe>
      <el-table-column label="Time" width="170"><template #default="scope">{{ formatTimestamp(scope.row.created_at) }}</template></el-table-column>
      <el-table-column label="Actor / action" min-width="230"><template #default="scope"><strong>{{ scope.row.actor_username }}</strong><div>{{ scope.row.action }}</div><small>{{ scope.row.entity_type }} #{{ scope.row.entity_id }}</small></template></el-table-column>
      <el-table-column label="Before" min-width="270"><template #default="scope"><pre>{{ formatSummary(scope.row.before_summary) }}</pre></template></el-table-column>
      <el-table-column label="After" min-width="300"><template #default="scope"><pre>{{ formatSummary(scope.row.after_summary) }}</pre></template></el-table-column>
      <el-table-column label="Request ID" min-width="250"><template #default="scope"><code>{{ scope.row.request_id }}</code></template></el-table-column>
    </el-table>
  </section>
</template>
