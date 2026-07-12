<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const connected = ref(false)
const token = ref('')
const loading = ref(true)
const busy = ref(false)
const error = ref('')
const notice = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    const s = await api.get('/api/settings')
    connected.value = s.lunch_money_connected
  } catch (e) {
    if (e.status === 401) return router.push('/login')
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function save() {
  error.value = ''
  notice.value = ''
  if (!token.value.trim()) {
    error.value = 'Paste your Lunch Money access token'
    return
  }
  busy.value = true
  try {
    const s = await api.put('/api/settings', { lunch_money_token: token.value.trim() })
    connected.value = s.lunch_money_connected
    token.value = ''
    notice.value = 'Lunch Money connected'
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function disconnect() {
  if (!confirm('Disconnect Lunch Money? New transactions will stop syncing.')) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const s = await api.put('/api/settings', { lunch_money_token: '' })
    connected.value = s.lunch_money_connected
    notice.value = 'Disconnected'
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink class="link" style="color: #fff" to="/">← Back</RouterLink>
      <h1>Settings</h1>
      <span style="width: 40px"></span>
    </header>

    <main class="content">
      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="notice" class="notice">{{ notice }}</div>

      <p v-if="loading" class="muted">Loading…</p>

      <div v-else class="card" style="padding: 16px">
        <div class="section-title">Lunch Money</div>
        <p class="muted" style="margin-top: 0">
          Paste an access token from your Lunch Money
          <em>Developers</em> page. New transactions sync automatically every hour into
          <RouterLink class="inline-link" to="/lunchmoney">Review</RouterLink>.
        </p>

        <div class="status" :class="connected ? 'on' : 'off'">
          {{ connected ? '● Connected' : '○ Not connected' }}
        </div>

        <div class="field">
          <label>{{ connected ? 'Replace token' : 'Access token' }}</label>
          <input v-model="token" type="password" autocomplete="off" placeholder="Paste token" />
        </div>
        <button class="btn" :disabled="busy" @click="save">
          {{ connected ? 'Update token' : 'Connect' }}
        </button>
        <button v-if="connected" class="btn danger" :disabled="busy" @click="disconnect">
          Disconnect
        </button>
      </div>
    </main>
  </div>
</template>

<style scoped>
.status {
  font-weight: 700;
  margin: 8px 0 16px;
}

.status.on {
  color: var(--accent-dark);
}

.status.off {
  color: var(--muted);
}

.inline-link {
  color: var(--accent-dark);
  font-weight: 600;
}
</style>
