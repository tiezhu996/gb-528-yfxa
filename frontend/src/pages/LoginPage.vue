<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowRight, ClipboardCheck, KeyRound } from 'lucide-vue-next'
import { ElMessage } from 'element-plus'
import SafetyBoundary from '../components/common/SafetyBoundary.vue'
import { errorMessage } from '../api/client'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const error = ref('')
const form = reactive({ username: 'programmer', password: 'programmer123' })

function choose(role: 'programmer' | 'reviewer') {
  form.username = role
  form.password = role === 'programmer' ? 'programmer123' : 'reviewer123'
}

async function submit() {
  if (auth.loading) return
  error.value = ''
  try {
    await auth.signIn(form.username, form.password)
    ElMessage.success(`Signed in as ${auth.user?.role.replaceAll('_', ' ')}`)
    await router.replace('/devices')
  } catch (cause) {
    error.value = errorMessage(cause)
  }
}
</script>

<template>
  <main class="login-shell">
    <section class="login-identity">
      <div class="brand-block large"><span class="brand-mark">RQ</span><div><strong>Rigging Cue Desk</strong><span>Offline interlock review</span></div></div>
      <div class="cue-sheet-mark">
        <p class="eyebrow">REHEARSAL WORKSPACE · REV 01</p>
        <h1>Stage rigging<br />cue interlock</h1>
        <p>Versioned motion assumptions, collision windows, rule evidence, and named human review for theatre technical teams.</p>
      </div>
      <SafetyBoundary />
    </section>
    <section class="login-panel">
      <div class="login-form-wrap">
        <KeyRound :size="24" />
        <div><p class="eyebrow">CONTROLLED ACCESS</p><h2>Open the rehearsal desk</h2></div>
        <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon />
        <el-form label-position="top">
          <el-form-item label="Username"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
          <el-form-item label="Password"><el-input v-model="form.password" type="password" show-password autocomplete="current-password" @keyup.enter="submit" /></el-form-item>
          <el-button type="primary" :loading="auth.loading" :icon="ArrowRight" native-type="button" @click="submit">Sign in</el-button>
        </el-form>
        <div class="account-switch">
          <span>Test roles</span>
          <el-button text @click="choose('programmer')">Programmer</el-button>
          <el-button text :icon="ClipboardCheck" @click="choose('reviewer')">Safety reviewer</el-button>
        </div>
      </div>
    </section>
  </main>
</template>
