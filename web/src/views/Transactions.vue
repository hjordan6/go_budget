<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const txns = ref([])
const buckets = ref([])
const loading = ref(true)
const error = ref('')

const money = new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD' })
const dateFmt = new Intl.DateTimeFormat(undefined, {
  month: 'short',
  day: 'numeric',
  year: 'numeric',
})

const bucketName = computed(() => {
  const map = {}
  for (const b of buckets.value) map[b.id] = b.name
  return (id) => map[id] || `#${id}`
})

// Group transactions by calendar day for a scannable, dated list.
const groups = computed(() => {
  const out = []
  let current = null
  for (const t of txns.value) {
    const day = dateFmt.format(new Date(t.created_at))
    if (!current || current.day !== day) {
      current = { day, items: [] }
      out.push(current)
    }
    current.items.push(t)
  }
  return out
})

function signed(a) {
  return (a < 0 ? '-' : '+') + money.format(Math.abs(a))
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [t, b] = await Promise.all([
      api.get('/api/transactions'),
      api.get('/api/buckets'),
    ])
    txns.value = t
    buckets.value = b
  } catch (e) {
    if (e.status === 401) return router.push('/login')
    error.value = e.message
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink class="link" style="color: #fff" to="/">← Back</RouterLink>
      <h1>Transactions</h1>
      <span style="width: 40px"></span>
    </header>

    <main class="content">
      <div v-if="error" class="error">{{ error }}</div>

      <p v-if="loading" class="muted">Loading…</p>

      <div v-else-if="txns.length === 0" class="empty">
        No transactions yet.<br />Open a bucket to add one.
      </div>

      <div v-else>
        <div v-for="g in groups" :key="g.day" class="day-group">
          <div class="day-label">{{ g.day }}</div>
          <div class="card">
            <div v-for="t in g.items" :key="t.id" class="txn-row">
              <div class="txn-main">
                <div class="txn-desc">{{ t.description || '—' }}</div>
                <div class="txn-bucket">{{ bucketName(t.bucket_id) }}</div>
              </div>
              <div class="txn-amount" :class="t.amount < 0 ? 'neg' : 'pos'">
                {{ signed(t.amount) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.day-group {
  margin-bottom: 18px;
}

.day-label {
  font-size: 0.8rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted);
  margin: 0 4px 8px;
}

.card {
  padding: 4px 16px;
}

.txn-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}

.txn-row:last-child {
  border-bottom: none;
}

.txn-main {
  flex: 1;
  min-width: 0;
}

.txn-desc {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.txn-bucket {
  font-size: 0.8rem;
  color: var(--muted);
  margin-top: 2px;
}

.txn-amount {
  font-weight: 700;
  white-space: nowrap;
}

.txn-amount.neg {
  color: var(--danger);
}

.txn-amount.pos {
  color: var(--accent-dark);
}
</style>
