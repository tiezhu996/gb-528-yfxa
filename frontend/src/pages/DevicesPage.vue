<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { Check, Pencil, Plus, RotateCcw, Save } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import PageHeader from '../components/common/PageHeader.vue'
import { errorMessage } from '../api/client'
import { useAuth } from '../hooks/useAuth'
import { useDeviceStore } from '../stores/devices'
import type { CreateDeviceInput, DeviceStatus, DeviceType, RiggingDevice } from '../types/device'

const store = useDeviceStore()
const { canProgram } = useAuth()
const selectedId = ref<number | null>(null)
const saving = ref(false)
const localError = ref('')
const selected = computed(() => store.items.find((item) => item.id === selectedId.value) ?? null)
const form = reactive({ device_code: '', name: '', device_type: 'motorized_batten' as DeviceType, max_load_kg: 500, max_speed_ms: 0.4, travel_min_m: 4, travel_max_m: 16, safety_zone: 'overstage-c', device_status: 'available' as DeviceStatus })

function reset() {
  selectedId.value = null
  Object.assign(form, { device_code: '', name: '', device_type: 'motorized_batten', max_load_kg: 500, max_speed_ms: 0.4, travel_min_m: 4, travel_max_m: 16, safety_zone: 'overstage-c', device_status: 'available' })
  localError.value = ''
}

function edit(item: RiggingDevice) {
  selectedId.value = item.id
  Object.assign(form, { device_code: item.device_code, name: item.name, device_type: item.device_type, max_load_kg: item.max_load_kg, max_speed_ms: item.max_speed_ms, travel_min_m: item.travel_min_m, travel_max_m: item.travel_max_m, safety_zone: item.safety_zone, device_status: item.device_status })
}

function selectDevice(item: RiggingDevice | undefined) {
  if (item) edit(item)
}

async function save() {
  if (saving.value) return
  saving.value = true
  localError.value = ''
  try {
    if (selected.value) {
      await store.update(selected.value.id, { name: form.name, device_type: form.device_type, max_load_kg: form.max_load_kg, max_speed_ms: form.max_speed_ms, travel_min_m: form.travel_min_m, travel_max_m: form.travel_max_m, safety_zone: form.safety_zone, device_status: form.device_status, version: selected.value.version })
      ElMessage.success('Device limits updated and audited')
    } else {
      await store.create({ ...form } as CreateDeviceInput)
      ElMessage.success('Device model created')
      reset()
    }
  } catch (cause) {
    localError.value = errorMessage(cause)
  } finally {
    saving.value = false
  }
}

onMounted(() => store.load().catch(() => undefined))
</script>

<template>
  <PageHeader eyebrow="EQUIPMENT MODEL / VERSIONED LIMITS" title="Rigging devices" description="Maintain offline motion envelopes and inspect the enabled interlocks that apply to each carrier.">
    <el-button v-if="canProgram" :icon="Plus" @click="reset">New device</el-button>
  </PageHeader>
  <el-alert v-if="store.error || localError" :title="localError || store.error" type="error" :closable="false" show-icon />
  <div class="split-workspace device-workspace">
    <section class="data-section">
      <div class="section-heading"><div><p class="eyebrow">REGISTER</p><h2>{{ store.items.length }} modeled devices</h2></div><span class="legend"><Check :size="14" /> limits are planning assumptions</span></div>
      <el-table :data="store.items" :loading="store.loading" stripe highlight-current-row @current-change="selectDevice">
        <el-table-column prop="device_code" label="Device" min-width="150"><template #default="scope"><strong>{{ scope.row.device_code }}</strong><div class="subtle">{{ scope.row.name }}</div></template></el-table-column>
        <el-table-column prop="device_type" label="Type" min-width="145" />
        <el-table-column label="Travel" width="120"><template #default="scope">{{ scope.row.travel_min_m }}–{{ scope.row.travel_max_m }} m</template></el-table-column>
        <el-table-column label="Load / speed" min-width="145"><template #default="scope">{{ scope.row.max_load_kg }} kg<div class="subtle">{{ scope.row.max_speed_ms }} m/s</div></template></el-table-column>
        <el-table-column prop="safety_zone" label="Safety zone" min-width="130" />
        <el-table-column label="Rules" width="86"><template #default="scope"><el-tag effect="plain">{{ scope.row.applicable_rules.length }}</el-tag></template></el-table-column>
        <el-table-column label="Status" width="130"><template #default="scope"><span class="plain-status" :class="scope.row.device_status">{{ scope.row.device_status.replaceAll('_', ' ') }}</span></template></el-table-column>
        <el-table-column v-if="canProgram" width="56"><template #default="scope"><el-button circle text :icon="Pencil" title="Edit device limits" @click.stop="edit(scope.row)" /></template></el-table-column>
      </el-table>
    </section>
    <aside class="editor-panel">
      <div class="section-heading"><div><p class="eyebrow">{{ selected ? `REVISION ${selected.version}` : 'NEW RECORD' }}</p><h2>{{ selected ? selected.device_code : 'Device envelope' }}</h2></div><el-button v-if="selected" circle text :icon="RotateCcw" title="Clear selection" @click="reset" /></div>
      <template v-if="canProgram">
        <el-form label-position="top">
          <div class="form-grid two"><el-form-item label="Device code"><el-input v-model="form.device_code" :disabled="Boolean(selected)" placeholder="BATTEN-04" /></el-form-item><el-form-item label="Name"><el-input v-model="form.name" /></el-form-item></div>
          <div class="form-grid two"><el-form-item label="Device type"><el-select v-model="form.device_type"><el-option v-for="value in ['motorized_batten','motorized_bridge','scenic_carrier','point_hoist','manual_counterweight']" :key="value" :value="value" :label="value.replaceAll('_',' ')" /></el-select></el-form-item><el-form-item label="Status"><el-select v-model="form.device_status"><el-option v-for="value in ['available','inspection_hold','retired']" :key="value" :value="value" :label="value.replaceAll('_',' ')" /></el-select></el-form-item></div>
          <div class="form-grid two"><el-form-item label="Maximum load (kg)"><el-input-number v-model="form.max_load_kg" :min="1" :max="100000" /></el-form-item><el-form-item label="Maximum speed (m/s)"><el-input-number v-model="form.max_speed_ms" :min="0.01" :max="10" :step="0.05" :precision="2" /></el-form-item></div>
          <div class="form-grid two"><el-form-item label="Travel minimum (m)"><el-input-number v-model="form.travel_min_m" :min="-20" :max="200" :step="0.5" /></el-form-item><el-form-item label="Travel maximum (m)"><el-input-number v-model="form.travel_max_m" :min="-20" :max="200" :step="0.5" /></el-form-item></div>
          <el-form-item label="Safety zone"><el-input v-model="form.safety_zone" placeholder="overstage-c" /></el-form-item>
          <el-button type="primary" native-type="button" :icon="Save" :loading="saving" @click="save">{{ selected ? 'Save revision' : 'Create device' }}</el-button>
        </el-form>
      </template>
      <template v-else>
        <div class="read-only-note">Safety reviewers can inspect device limits and applicable rules. Device model changes require a programmer or administrator.</div>
      </template>
      <div v-if="selected" class="rule-index">
        <p class="eyebrow">APPLICABLE RULES</p>
        <div v-if="selected.applicable_rules.length === 0" class="empty-inline">No scoped rules.</div>
        <div v-for="rule in selected.applicable_rules" :key="rule.id" class="rule-reference"><strong>{{ rule.rule_code }}</strong><span>{{ rule.rule_type.replaceAll('_', ' ') }} · v{{ rule.rule_version }}</span><em :class="rule.severity">{{ rule.enabled ? rule.severity : 'disabled' }}</em></div>
      </div>
    </aside>
  </div>
</template>
