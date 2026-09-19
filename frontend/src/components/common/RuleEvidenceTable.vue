<script setup lang="ts">
import { AlertTriangle, Ban, Check, CircleHelp } from 'lucide-vue-next'
import type { RuleEvidence } from '../../types/interlock'
import { formatMillis } from '../../utils/timeline'

defineProps<{ evidence: RuleEvidence[]; loading?: boolean }>()

const resultType = (result: string) => ({ pass: 'success', warning: 'warning', blocker: 'danger', invalid: 'info' }[result] ?? 'info')
const resultIcon = (result: string) => ({ pass: Check, warning: AlertTriangle, blocker: Ban, invalid: CircleHelp }[result] ?? CircleHelp)
</script>

<template>
  <el-table :data="evidence" :loading="loading" stripe class="evidence-table" empty-text="No rule evidence in this snapshot">
    <el-table-column label="Result" width="122">
      <template #default="scope">
        <el-tag :type="resultType(scope.row.result)" effect="plain" class="result-tag">
          <component :is="resultIcon(scope.row.result)" :size="14" />{{ scope.row.result }}
        </el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="rule_code" label="Rule" min-width="130" />
    <el-table-column label="Cue / device" min-width="190">
      <template #default="scope">
        <strong>{{ scope.row.cue_codes?.join(', ') || 'Rule set' }}</strong>
        <div class="subtle">{{ scope.row.device_codes?.join(', ') || 'No device scope' }}</div>
      </template>
    </el-table-column>
    <el-table-column label="Window" width="145">
      <template #default="scope">{{ formatMillis(scope.row.window_start_ms) }}–{{ formatMillis(scope.row.window_end_ms) }}</template>
    </el-table-column>
    <el-table-column label="Actual / threshold" min-width="160">
      <template #default="scope"><strong>{{ scope.row.actual_value }}</strong> / {{ scope.row.threshold_value }} {{ scope.row.unit }}</template>
    </el-table-column>
    <el-table-column prop="message" label="Evidence" min-width="300" />
  </el-table>
</template>
