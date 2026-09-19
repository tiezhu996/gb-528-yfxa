<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { FlaskConical, Pencil, Plus, RotateCcw, Save, ShieldCheck } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import PageHeader from '../components/common/PageHeader.vue'
import RuleEvidenceTable from '../components/common/RuleEvidenceTable.vue'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useDeviceStore } from '../stores/devices'
import { useRehearsalStore } from '../stores/rehearsals'
import { useRuleStore } from '../stores/rules'
import type { InterlockRule, RuleEvidence, RuleThreshold, RuleType } from '../types/interlock'

const rules = useRuleStore()
const devices = useDeviceStore()
const runs = useRehearsalStore()
const { canProgram, canReview } = useAuth()
const selectedId = ref<number | null>(null)
const saving = ref(false)
const localError = ref('')
const actualValue = ref(0)
const selected = computed(() => rules.items.find((item) => item.id === selectedId.value) ?? null)
const form = reactive({ rule_code: '', rule_type: 'dependency_guard' as RuleType, device_ids: [] as number[], severity: 'blocker' as 'warning' | 'blocker', enabled: true, explanation: '', use_device_limits: false, primary_threshold: 0, secondary_threshold: 0 })

const evidence = computed<RuleEvidence[]>(() => {
  if (!selected.value) return []
  const latest = runs.items[0]?.rule_results.filter((item) => item.rule_code === selected.value?.rule_code) ?? []
  const tested = rules.tests[selected.value.id]
  if (!tested) return latest
  return [{ rule_code: tested.rule_code, rule_type: tested.rule_type, result: tested.result, severity: selected.value.severity, cue_codes: [], device_codes: [], window_start_ms: 0, window_end_ms: 0, actual_value: tested.actual_value, threshold_value: tested.threshold_value, unit: tested.unit, message: tested.boundary }, ...latest]
})

function reset() {
  selectedId.value = null
  Object.assign(form, { rule_code: '', rule_type: 'dependency_guard', device_ids: [], severity: 'blocker', enabled: true, explanation: '', use_device_limits: false, primary_threshold: 0, secondary_threshold: 0 })
  localError.value = ''
}

function edit(rule: InterlockRule) {
  selectedId.value = rule.id
  Object.assign(form, { rule_code: rule.rule_code, rule_type: rule.rule_type, device_ids: [...(rule.device_ids ?? [])], severity: rule.severity, enabled: rule.enabled, explanation: rule.explanation, use_device_limits: Boolean(rule.threshold.use_device_limits), primary_threshold: thresholdPrimary(rule), secondary_threshold: rule.threshold.max_position_m ?? 0 })
}

function thresholdPrimary(rule: InterlockRule): number {
  return rule.threshold.max_load_kg ?? rule.threshold.max_speed_ms ?? rule.threshold.min_position_m ?? rule.threshold.minimum_gap_ms ?? 0
}

function threshold(): RuleThreshold {
  if (form.use_device_limits && ['load_limit', 'speed_limit', 'travel_limit'].includes(form.rule_type)) return { use_device_limits: true }
  if (form.rule_type === 'load_limit') return { max_load_kg: form.primary_threshold }
  if (form.rule_type === 'speed_limit') return { max_speed_ms: form.primary_threshold }
  if (form.rule_type === 'travel_limit') return { min_position_m: form.primary_threshold, max_position_m: form.secondary_threshold }
  return { minimum_gap_ms: form.primary_threshold }
}

async function save() {
  if (saving.value) return
  saving.value = true
  localError.value = ''
  try {
    if (selected.value) {
      await rules.update(selected.value.id, { device_ids: form.device_ids, threshold: threshold(), severity: form.severity, enabled: selected.value.enabled, explanation: form.explanation, rule_version: selected.value.rule_version })
      ElMessage.success('Rule revision saved and audited')
    } else {
      const created = await rules.create({ rule_code: form.rule_code, rule_type: form.rule_type, device_ids: form.device_ids, threshold: threshold(), severity: form.severity, enabled: form.enabled, explanation: form.explanation })
      edit(created)
      ElMessage.success('Interlock rule created')
    }
  } catch (cause) {
    localError.value = errorMessage(cause)
  } finally {
    saving.value = false
  }
}

async function toggle(rule: InterlockRule, enabled: boolean) {
  localError.value = ''
  try {
    const updated = await rules.toggle(rule, enabled)
    edit(updated)
    ElMessage.success(`${updated.rule_code} ${updated.enabled ? 'enabled' : 'disabled'}`)
  } catch (cause) {
    localError.value = errorMessage(cause)
  }
}

function onToggle(rule: InterlockRule, value: string | number | boolean) {
  return toggle(rule, Boolean(value))
}

async function test() {
  if (!selected.value) return
  localError.value = ''
  try {
    const result = await rules.test(selected.value, actualValue.value)
    ElMessage.success(`Threshold test returned ${result.result}`)
  } catch (cause) {
    localError.value = errorMessage(cause)
  }
}

onMounted(() => Promise.all([rules.load(), devices.load(), runs.load()]).catch(() => undefined))
</script>

<template>
  <PageHeader eyebrow="RULE LIBRARY / VERSIONED THRESHOLDS" title="Interlock rules" description="Scope physical and dependency checks to modeled devices, test review thresholds, and retain every enabled-state revision.">
    <el-button v-if="canProgram" :icon="Plus" @click="reset">New rule</el-button>
  </PageHeader>
  <el-alert v-if="rules.error || devices.error || runs.error || localError" :title="localError || rules.error || devices.error || runs.error" type="error" :closable="false" show-icon />
  <div class="split-workspace rules-workspace">
    <section class="data-section rule-list">
      <div class="section-heading"><div><p class="eyebrow">ACTIVE LIBRARY</p><h2>{{ rules.items.filter((item) => item.enabled).length }} / {{ rules.items.length }} rules enabled</h2></div><ShieldCheck :size="20" /></div>
      <button v-for="rule in rules.items" :key="rule.id" class="rule-row" :class="{ selected: selectedId === rule.id }" @click="edit(rule)">
        <span class="rule-code">{{ rule.rule_code }}</span>
        <span><strong>{{ rule.rule_type.replaceAll('_', ' ') }}</strong><small>{{ rule.explanation }}</small></span>
        <em :class="rule.severity">{{ rule.severity }}</em>
        <el-switch v-if="canReview" :model-value="rule.enabled" :aria-label="`Toggle ${rule.rule_code}`" @click.stop @change="onToggle(rule, $event)" />
        <span v-else class="plain-status" :class="rule.enabled ? 'available' : 'retired'">{{ rule.enabled ? 'enabled' : 'disabled' }}</span>
      </button>
    </section>
    <aside class="editor-panel">
      <div class="section-heading"><div><p class="eyebrow">{{ selected ? `RULE VERSION ${selected.rule_version}` : 'NEW RULE' }}</p><h2>{{ selected ? selected.rule_code : 'Rule definition' }}</h2></div><el-button v-if="selected" circle text :icon="RotateCcw" title="Clear selection" @click="reset" /></div>
      <el-form v-if="canProgram" label-position="top">
        <div class="form-grid two"><el-form-item label="Rule code"><el-input v-model="form.rule_code" :disabled="Boolean(selected)" placeholder="ZONE-C-02" /></el-form-item><el-form-item label="Rule type"><el-select v-model="form.rule_type" :disabled="Boolean(selected)"><el-option v-for="value in ['load_limit','speed_limit','travel_limit','zone_exclusion','dependency_guard']" :key="value" :value="value" :label="value.replaceAll('_',' ')" /></el-select></el-form-item></div>
        <el-form-item label="Device scope"><el-select v-model="form.device_ids" multiple clearable><el-option v-for="device in devices.items" :key="device.id" :value="device.id" :label="`${device.device_code} · ${device.safety_zone}`" /></el-select></el-form-item>
        <div class="form-grid two"><el-form-item label="Severity"><el-segmented v-model="form.severity" :options="['warning','blocker']" /></el-form-item><el-form-item v-if="!selected" label="Initial state"><el-switch v-model="form.enabled" active-text="Enabled" inactive-text="Disabled" /></el-form-item></div>
        <el-checkbox v-if="['load_limit','speed_limit','travel_limit'].includes(form.rule_type)" v-model="form.use_device_limits">Use each device envelope</el-checkbox>
        <div v-if="!form.use_device_limits" class="form-grid two"><el-form-item :label="form.rule_type === 'dependency_guard' ? 'Minimum gap (ms)' : form.rule_type === 'zone_exclusion' ? 'Allowed overlap (ms)' : form.rule_type === 'speed_limit' ? 'Maximum speed (m/s)' : form.rule_type === 'load_limit' ? 'Maximum load (kg)' : 'Minimum position (m)'"><el-input-number v-model="form.primary_threshold" :step="form.rule_type === 'speed_limit' ? 0.05 : 1" /></el-form-item><el-form-item v-if="form.rule_type === 'travel_limit'" label="Maximum position (m)"><el-input-number v-model="form.secondary_threshold" :step="0.5" /></el-form-item></div>
        <el-form-item label="Explanation"><el-input v-model="form.explanation" type="textarea" :rows="3" maxlength="600" show-word-limit /></el-form-item>
        <el-button type="primary" native-type="button" :icon="Save" :loading="saving" @click="save">{{ selected ? 'Save rule revision' : 'Create rule' }}</el-button>
      </el-form>
      <div v-if="selected" class="threshold-test">
        <p class="eyebrow">THRESHOLD PROBE</p>
        <div class="inline-control"><el-input-number v-model="actualValue" /><el-button :icon="FlaskConical" :disabled="selected.rule_type === 'travel_limit' || Boolean(selected.threshold.use_device_limits)" @click="test">Test actual value</el-button></div>
        <p class="boundary-copy">A probe evaluates the stored threshold only. Full spatial and device-limit evidence comes from an immutable rehearsal snapshot.</p>
      </div>
    </aside>
  </div>
  <section class="data-section evidence-section">
    <div class="section-heading"><div><p class="eyebrow">LATEST EVIDENCE</p><h2>{{ selected ? selected.rule_code : 'Select a rule' }}</h2></div><span>{{ evidence.length }} result rows</span></div>
    <RuleEvidenceTable :evidence="evidence" />
  </section>
</template>
