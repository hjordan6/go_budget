<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()

const tab = ref('pending') // pending | imported | ignored
const rows = ref([])
const buckets = ref([])
const loading = ref(true)
const error = ref('')
const notice = ref('')
const syncing = ref(false)
// Per-row busy + chosen bucket, keyed by row id (reactive so v-model tracks new keys).
const busy = reactive({})
const bucketFor = reactive({})

const money = new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD' })
const dateFmt = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric', year: 'numeric' })

function fmtDate(d) {
  // Lunch Money dates are plain YYYY-MM-DD; parse as local, not UTC.
  const [y, m, day] = d.split('-').map(Number)
  return dateFmt.format(new Date(y, m - 1, day))
}

function signed(a) {
  return (a < 0 ? '-' : '+') + money.format(Math.abs(a))
}

function bucketName(id) {
  const b = buckets.value.find((x) => x.id === id)
  return b ? b.name : `#${id}`
}

async function loadBuckets() {
  buckets.value = await api.get('/api/buckets')
}

async function loadRows() {
  loading.value = true
  error.value = ''
  try {
    rows.value = await api.get(`/api/lunchmoney/transactions?status=${tab.value}`)
    for (const t of rows.value) {
      if (bucketFor[t.id] === undefined) {
        bucketFor[t.id] = buckets.value.length ? buckets.value[0].id : null
      }
    }
  } catch (e) {
    if (e.status === 401) return router.push('/login')
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function switchTab(t) {
  if (tab.value === t) return
  tab.value = t
  notice.value = ''
  await loadRows()
}

async function syncNow() {
  syncing.value = true
  error.value = ''
  notice.value = ''
  try {
    const res = await api.post('/api/lunchmoney/sync')
    notice.value = res.added > 0 ? `Pulled ${res.added} new transaction(s)` : 'Up to date — nothing new'
    if (tab.value === 'pending') await loadRows()
  } catch (e) {
    error.value = e.message
  } finally {
    syncing.value = false
  }
}

function removeRow(id) {
  rows.value = rows.value.filter((t) => t.id !== id)
}

async function importRow(t) {
  error.value = ''
  const bucketId = bucketFor[t.id]
  if (!bucketId) {
    error.value = 'Pick a bucket first'
    return
  }
  const amount = parseFloat(t.amount)
  if (!(amount !== 0)) {
    error.value = 'Amount cannot be zero'
    return
  }
  busy[t.id] = true
  try {
    await api.post(`/api/lunchmoney/transactions/${t.id}/import`, {
      bucket_id: bucketId,
      payee: (t.payee || '').trim(),
      amount,
    })
    removeRow(t.id)
    notice.value = 'Imported'
  } catch (e) {
    error.value = e.message
  } finally {
    busy[t.id] = false
  }
}

async function ignoreRow(t) {
  busy[t.id] = true
  error.value = ''
  try {
    await api.post(`/api/lunchmoney/transactions/${t.id}/ignore`)
    removeRow(t.id)
    notice.value = 'Marked as accounted for'
  } catch (e) {
    error.value = e.message
  } finally {
    busy[t.id] = false
  }
}

async function restoreRow(t) {
  busy[t.id] = true
  error.value = ''
  try {
    await api.post(`/api/lunchmoney/transactions/${t.id}/restore`)
    removeRow(t.id)
    notice.value = 'Moved back to review'
  } catch (e) {
    error.value = e.message
  } finally {
    busy[t.id] = false
  }
}

onMounted(async () => {
  try {
    await loadBuckets()
  } catch (e) {
    if (e.status === 401) return router.push('/login')
  }
  await loadRows()
})
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink class="link" style="color: #fff" to="/">← Back</RouterLink>
      <h1>Review</h1>
      <RouterLink class="link" style="color: #fff" to="/settings">⚙</RouterLink>
    </header>

    <main class="content">
      <div class="tabs">
        <button class="tab" :class="{ active: tab === 'pending' }" @click="switchTab('pending')">To review</button>
        <button class="tab" :class="{ active: tab === 'imported' }" @click="switchTab('imported')">Imported</button>
        <button class="tab" :class="{ active: tab === 'ignored' }" @click="switchTab('ignored')">Ignored</button>
      </div>

      <button class="btn secondary sync-btn" :disabled="syncing" @click="syncNow">
        {{ syncing ? 'Syncing…' : '↻ Sync from Lunch Money' }}
      </button>

      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="notice" class="notice">{{ notice }}</div>

      <p v-if="loading" class="muted">Loading…</p>

      <div v-else-if="rows.length === 0" class="empty">
        <template v-if="tab === 'pending'">
          Nothing to review.<br />New Lunch Money transactions show up here.
        </template>
        <template v-else-if="tab === 'imported'">No imported transactions yet.</template>
        <template v-else>Nothing ignored.</template>
      </div>

      <!-- Pending: editable review cards -->
      <div v-else-if="tab === 'pending'">
        <div v-for="t in rows" :key="t.id" class="card review-card">
          <div class="meta">
            <span>{{ fmtDate(t.date) }}</span>
            <span class="amt" :class="parseFloat(t.amount) < 0 ? 'neg' : 'pos'">{{ signed(parseFloat(t.amount)) }}</span>
          </div>
          <div class="field">
            <label>Name</label>
            <input v-model="t.payee" type="text" placeholder="Payee" />
          </div>
          <div class="field">
            <label>Amount (negative = expense)</label>
            <input v-model="t.amount" type="number" inputmode="decimal" step="0.01" />
          </div>
          <div class="field">
            <label>Bucket</label>
            <select v-model="bucketFor[t.id]" class="select">
              <option v-for="b in buckets" :key="b.id" :value="b.id">{{ b.name }}</option>
            </select>
          </div>
          <div class="review-actions">
            <button class="btn" :disabled="busy[t.id]" @click="importRow(t)">Import</button>
            <button class="btn ghost" :disabled="busy[t.id]" @click="ignoreRow(t)">Accounted for</button>
          </div>
        </div>
      </div>

      <!-- Imported / ignored: read-only rows -->
      <div v-else class="card">
        <div v-for="t in rows" :key="t.id" class="done-row">
          <div class="done-main">
            <div class="done-name">{{ t.payee || '—' }}</div>
            <div class="done-sub">
              {{ fmtDate(t.date) }}
              <template v-if="tab === 'imported' && t.transaction_id"> · imported</template>
            </div>
          </div>
          <div class="done-amt" :class="t.amount < 0 ? 'neg' : 'pos'">{{ signed(t.amount) }}</div>
          <button
            v-if="tab === 'ignored'"
            class="icon-btn"
            aria-label="Move back to review"
            :disabled="busy[t.id]"
            @click="restoreRow(t)"
          >
            ↩
          </button>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 0;
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
  margin-bottom: 12px;
}

.tab {
  flex: 1 1 0;
  padding: 10px;
  font: inherit;
  font-weight: 600;
  border: none;
  background: var(--surface);
  color: var(--muted);
  cursor: pointer;
}

.tab.active {
  background: var(--accent);
  color: #fff;
}

.sync-btn {
  width: 100%;
  margin-bottom: 14px;
}

.review-card {
  padding: 16px;
  margin-bottom: 14px;
}

.meta {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
  font-size: 0.85rem;
  color: var(--muted);
}

.amt {
  font-weight: 700;
  font-size: 1rem;
}

.amt.neg,
.done-amt.neg {
  color: var(--danger);
}

.amt.pos,
.done-amt.pos {
  color: var(--accent-dark);
}

.review-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.review-actions .btn {
  flex: 1;
}

.done-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}

.done-row:last-child {
  border-bottom: none;
}

.done-main {
  flex: 1;
  min-width: 0;
}

.done-name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.done-sub {
  font-size: 0.8rem;
  color: var(--muted);
  margin-top: 2px;
}

.done-amt {
  font-weight: 700;
  white-space: nowrap;
}
</style>
