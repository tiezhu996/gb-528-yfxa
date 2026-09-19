<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { ArrowRight, CheckCircle2, LockKeyhole, Pencil, Plus, RotateCcw, Save, Trash2, Undo2 } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import PageHeader from '../components/common/PageHeader.vue'
import CueStatusBadge from '../components/common/CueStatusBadge.vue'
import TimelineTrack from '../components/common/TimelineTrack.vue'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useCueStore } from '../stores/cues'
import { useDeviceStore } from '../stores/devices'
import type { CueAction, CueDefinition } from '../types/cue'
import type { TimelineEvent } from '../types/rehearsal'

const cues = useCueStore()
const devices = useDeviceStore()
const { canProgram, canReview } = useAuth()
const selectedId = ref<number | null>(null)
const saving = ref(false)
const localError = ref('')
const reason = ref('Reviewed against the offline rehearsal assumptions and evidence.')
const selected = computed(() => cues.items.find((item) => item.id === selectedId.value) ?? null)
const form = reactive({ cue_code: '', name: '', sequence_no: 40, start_offset_ms: 36000, duration_ms: 8000, actions: [] as CueAction[], dependency_ids: [] as number[] })

const previewEvents = computed<TimelineEvent[]>(() => cues.items.flatMap((cue) => cue.actions.map((action) => {
  const device = devices.items.find((item) => item.id === action.device_id)
  return { cue_id: cue.id, cue_code: cue.cue_code, cue_sequence: cue.sequence_no, cue_version: cue.version, device_id: action.device_id, device_code: device?.device_code ?? `DEVICE-${action.device_id}`, device_name: device?.name ?? 'Unknown device', safety_zone: device?.safety_zone ?? '', start_ms: cue.start_offset_ms + action.start_offset_ms, end_ms: cue.start_offset_ms + action.start_offset_ms + action.duration_ms, from_position_m: action.from_position_m, to_position_m: action.to_position_m, load_kg: action.load_kg, speed_ms: Math.abs(action.to_position_m - action.from_position_m) / (action.duration_ms / 1000) }
})))

function blankAction(): CueAction {
  return { device_id: devices.items[0]?.id ?? 1, start_offset_ms: 0, duration_ms: 8000, from_position_m: 12, to_position_m: 9, load_kg: 250 }
}

function reset() {
  selectedId.value = null
  Object.assign(form, { cue_code: '', name: '', sequence_no: Math.max(10, ...cues.items.map((item) => item.sequence_no + 10)), start_offset_ms: Math.max(0, ...cues.items.map((item) => item.start_offset_ms + item.duration_ms + 1000)), duration_ms: 8000, actions: [blankAction()], dependency_ids: [] })
  localError.value = ''
}

function edit(cue: CueDefinition) {
  selectedId.value = cue.id
  Object.assign(form, { cue_code: cue.cue_code, name: cue.name, sequence_no: cue.sequence_no, start_offset_ms: cue.start_offset_ms, duration_ms: cue.duration_ms, actions: cue.actions.map((item) => ({ ...item })), dependency_ids: [...(cue.dependency_ids ?? [])] })
}

function selectCue(cue: CueDefinition | undefined) {
  if (cue) edit(cue)
}

function dependencyLabels(ids: number[] | null): string {
  return ids?.map((id) => cues.items.find((cue) => cue.id === id)?.cue_code ?? `#${id}`).join(', ') || 'None'
}

async function save() {
  if (saving.value) return
  saving.value = true
  localError.value = ''
  try {
    if (selected.value) {
      await cues.update(selected.value.id, { name: form.name, sequence_no: form.sequence_no, start_offset_ms: form.start_offset_ms, duration_ms: form.duration_ms, actions: form.actions, dependency_ids: form.dependency_ids, version: selected.value.version })
      ElMessage.success('Draft cue revision saved')
    } else {
      const created = await cues.create({ cue_code: form.cue_code, name: form.name, sequence_no: form.sequence_no, start_offset_ms: form.start_offset_ms, duration_ms: form.duration_ms, actions: form.actions, dependency_ids: form.dependency_ids })
      edit(created)
      ElMessage.success('Draft cue created')
    }
  } catch (cause) {
    localError.value = errorMessage(cause)
  } finally {
    saving.value = false
  }
}

async function transition(action: 'submit' | 'approve' | 'reject' | 'lock' | 'archive') {
  if (!selected.value) return
  localError.value = ''
  try {
    const updated = await cues.transition(selected.value.id, action, selected.value.version, reason.value)
    edit(updated)
    ElMessage.success(`Cue moved to ${updated.cue_status.replaceAll('_', ' ')}`)
  } catch (cause) {
    localError.value = errorMessage(cause)
  }
}

onMounted(async () => {
  await Promise.all([cues.load(), devices.load()]).catch(() => undefined)
  reset()
})
</script>

<template>
  <PageHeader eyebrow="CUE SET / ORDERED MOTION ASSUMPTIONS" title="Cue desk" description="Build versioned action intervals, declare predecessors, and route each cue through named review before locking it for rehearsal.">
    <el-button v-if="canProgram" :icon="Plus" @click="reset">New cue</el-button>
  </PageHeader>
  <el-alert v-if="cues.error || devices.error || localError" :title="localError || cues.error || devices.error" type="error" :closable="false" show-icon />
  <section class="data-section timeline-section">
    <div class="section-heading"><div><p class="eyebrow">CUE SET OVERVIEW</p><h2>Modeled motion lanes</h2></div><span>{{ cues.items.length }} cues · {{ previewEvents.length }} actions</span></div>
    <TimelineTrack :events="previewEvents" compact />
  </section>
  <div class="split-workspace cue-workspace">
    <section class="data-section">
      <el-table :data="cues.items" :loading="cues.loading" stripe highlight-current-row @current-change="selectCue">
        <el-table-column label="Cue" min-width="150"><template #default="scope"><strong>{{ scope.row.cue_code }}</strong><div class="subtle">{{ scope.row.name }}</div></template></el-table-column>
        <el-table-column prop="sequence_no" label="Seq" width="70" />
        <el-table-column label="Start / duration" width="145"><template #default="scope">{{ (scope.row.start_offset_ms / 1000).toFixed(1) }}s<div class="subtle">{{ (scope.row.duration_ms / 1000).toFixed(1) }}s duration</div></template></el-table-column>
        <el-table-column label="Actions" width="82"><template #default="scope">{{ scope.row.actions.length }}</template></el-table-column>
        <el-table-column label="Depends" min-width="120"><template #default="scope">{{ dependencyLabels(scope.row.dependency_ids) }}</template></el-table-column>
        <el-table-column label="Status" width="145"><template #default="scope"><CueStatusBadge :status="scope.row.cue_status" /></template></el-table-column>
        <el-table-column width="56"><template #default="scope"><el-button circle text :icon="Pencil" title="Inspect cue" @click.stop="edit(scope.row)" /></template></el-table-column>
      </el-table>
    </section>
    <aside class="editor-panel cue-editor">
      <div class="section-heading"><div><p class="eyebrow">{{ selected ? `CUE VERSION ${selected.version}` : 'NEW DRAFT' }}</p><h2>{{ selected ? selected.cue_code : 'Cue definition' }}</h2></div><el-button v-if="selected" circle text :icon="RotateCcw" title="Clear selection" @click="reset" /></div>
      <CueStatusBadge v-if="selected" :status="selected.cue_status" />
      <el-form v-if="canProgram && (!selected || selected.cue_status === 'draft')" label-position="top">
        <div class="form-grid two"><el-form-item label="Cue code"><el-input v-model="form.cue_code" :disabled="Boolean(selected)" placeholder="Q-040" /></el-form-item><el-form-item label="Name"><el-input v-model="form.name" /></el-form-item></div>
        <div class="form-grid three"><el-form-item label="Sequence"><el-input-number v-model="form.sequence_no" :min="1" /></el-form-item><el-form-item label="Start (ms)"><el-input-number v-model="form.start_offset_ms" :min="0" :step="100" /></el-form-item><el-form-item label="Duration (ms)"><el-input-number v-model="form.duration_ms" :min="100" :step="100" /></el-form-item></div>
        <el-form-item label="Dependencies"><el-select v-model="form.dependency_ids" multiple clearable><el-option v-for="cue in cues.items.filter((item) => item.id !== selectedId)" :key="cue.id" :label="`${cue.cue_code} · ${cue.name}`" :value="cue.id" /></el-select></el-form-item>
        <div class="action-editor-title"><strong>Device actions</strong><el-button text :icon="Plus" @click="form.actions.push(blankAction())">Add action</el-button></div>
        <div v-for="(action, index) in form.actions" :key="index" class="action-editor">
          <el-form-item label="Device"><el-select v-model="action.device_id"><el-option v-for="device in devices.items" :key="device.id" :label="`${device.device_code} · ${device.name}`" :value="device.id" /></el-select></el-form-item>
          <div class="form-grid three"><el-form-item label="Action start"><el-input-number v-model="action.start_offset_ms" :min="0" :step="100" /></el-form-item><el-form-item label="Action duration"><el-input-number v-model="action.duration_ms" :min="100" :step="100" /></el-form-item><el-form-item label="Load kg"><el-input-number v-model="action.load_kg" :min="0" :step="10" /></el-form-item></div>
          <div class="form-grid action-position"><el-form-item label="From m"><el-input-number v-model="action.from_position_m" :step="0.5" /></el-form-item><ArrowRight :size="16" /><el-form-item label="To m"><el-input-number v-model="action.to_position_m" :step="0.5" /></el-form-item><el-button v-if="form.actions.length > 1" circle text type="danger" :icon="Trash2" title="Remove action" @click="form.actions.splice(index,1)" /></div>
        </div>
        <el-button type="primary" native-type="button" :icon="Save" :loading="saving" @click="save">{{ selected ? 'Save draft revision' : 'Create draft cue' }}</el-button>
      </el-form>
      <div v-if="selected" class="transition-desk">
        <el-input v-model="reason" type="textarea" :rows="2" maxlength="500" show-word-limit />
        <div class="transition-actions">
          <el-button v-if="canProgram && selected.cue_status === 'draft'" type="primary" :icon="CheckCircle2" @click="transition('submit')">Submit review</el-button>
          <el-button v-if="canReview && selected.cue_status === 'pending_review'" type="success" :icon="CheckCircle2" @click="transition('approve')">Approve cue</el-button>
          <el-button v-if="canReview && selected.cue_status === 'pending_review'" :icon="Undo2" @click="transition('reject')">Return to draft</el-button>
          <el-button v-if="canReview && selected.cue_status === 'approved'" type="primary" :icon="LockKeyhole" @click="transition('lock')">Lock version</el-button>
          <el-button v-if="canReview && selected.cue_status === 'locked'" :icon="LockKeyhole" @click="transition('archive')">Archive</el-button>
        </div>
        <p class="boundary-copy">Approval and lock apply only to this offline cue version. They do not release or command machinery.</p>
      </div>
    </aside>
  </div>
</template>
