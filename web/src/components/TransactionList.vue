<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  bucketId: { type: Number, required: true },
})
// `changed` carries the refreshed buckets list so the parent can update balances.
const emit = defineEmits(['changed'])

const money = new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD' })
const dateFmt = new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' })

const txns = ref([])
const loading = ref(true)
const error = ref('')
const busy = ref(false)

// Add/edit form. `kind` toggles the sign of the stored amount: expenses are
// negative, deposits positive. `editingId` is null when adding a new one.
const editingId = ref(null)
const kind = ref('expense')
const amount = ref('')
const description = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    txns.value = await api.get(`/api/transactions?bucket_id=${props.bucketId}`)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = null
  kind.value = 'expense'
  amount.value = ''
  description.value = ''
}

function startEdit(t) {
  editingId.value = t.id
  kind.value = t.amount < 0 ? 'expense' : 'deposit'
  amount.value = String(Math.abs(t.amount))
  description.value = t.description
  error.value = ''
}

// signedAmount converts the positive input plus the kind toggle into the stored
// signed value.
function signedAmount() {
  const abs = Math.abs(parseFloat(amount.value))
  return kind.value === 'expense' ? -abs : abs
}

async function submit() {
  error.value = ''
  const abs = Math.abs(parseFloat(amount.value))
  if (!(abs > 0)) {
    error.value = 'Enter an amount greater than 0'
    return
  }
  const body = {
    bucket_id: props.bucketId,
    amount: signedAmount(),
    description: description.value.trim(),
  }
  busy.value = true
  try {
    let res
    if (editingId.value === null) {
      res = await api.post('/api/transactions', body)
    } else {
      res = await api.put(`/api/transactions/${editingId.value}`, body)
    }
    emit('changed', res.buckets)
    resetForm()
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function remove(t) {
  if (!confirm('Delete this transaction?')) return
  busy.value = true
  try {
    const res = await api.del(`/api/transactions/${t.id}`)
    emit('changed', res.buckets)
    if (editingId.value === t.id) resetForm()
    await load()
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

function signed(a) {
  return (a < 0 ? '-' : '+') + money.format(Math.abs(a))
}

onMounted(load)
</script>

<template>
  <div class="section">
    <div class="section-title">Transactions</div>

    <div v-if="error" class="error">{{ error }}</div>

    <!-- Add / edit form -->
    <div class="txn-form">
      <div class="seg">
        <button
          type="button"
          class="seg-btn"
          :class="{ active: kind === 'expense' }"
          @click="kind = 'expense'"
        >
          Expense
        </button>
        <button
          type="button"
          class="seg-btn"
          :class="{ active: kind === 'deposit' }"
          @click="kind = 'deposit'"
        >
          Deposit
        </button>
      </div>
      <div class="field">
        <label>Amount</label>
        <input
          v-model="amount"
          type="number"
          inputmode="decimal"
          step="0.01"
          min="0"
          placeholder="0.00"
        />
      </div>
      <div class="field">
        <label>Description</label>
        <input v-model="description" type="text" placeholder="e.g. Groceries" />
      </div>
      <button class="btn" :disabled="busy" @click="submit">
        {{ editingId === null ? 'Add transaction' : 'Save transaction' }}
      </button>
      <button v-if="editingId !== null" class="btn ghost" :disabled="busy" @click="resetForm">
        Cancel edit
      </button>
    </div>

    <!-- List -->
    <p v-if="loading" class="muted" style="margin-top: 14px">Loading…</p>
    <p v-else-if="txns.length === 0" class="muted" style="margin-top: 14px">
      No transactions yet.
    </p>
    <ul v-else class="txn-list">
      <li v-for="t in txns" :key="t.id" class="txn-row">
        <div class="txn-main">
          <div class="txn-desc">{{ t.description || '—' }}</div>
          <div class="txn-date">{{ dateFmt.format(new Date(t.created_at)) }}</div>
        </div>
        <div class="txn-amount" :class="t.amount < 0 ? 'neg' : 'pos'">{{ signed(t.amount) }}</div>
        <div class="txn-actions">
          <button class="icon-btn" aria-label="Edit" :disabled="busy" @click="startEdit(t)">✎</button>
          <button class="icon-btn" aria-label="Delete" :disabled="busy" @click="remove(t)">🗑</button>
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.txn-form {
  padding-bottom: 4px;
}

.seg {
  display: flex;
  gap: 0;
  margin-bottom: 16px;
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}

.seg-btn {
  flex: 1 1 0;
  padding: 12px;
  font: inherit;
  font-weight: 600;
  border: none;
  background: var(--surface);
  color: var(--muted);
  cursor: pointer;
}

.seg-btn.active {
  background: var(--accent);
  color: #fff;
}

.txn-list {
  list-style: none;
  margin: 14px 0 0;
  padding: 0;
}

.txn-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 0;
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

.txn-date {
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

.txn-actions {
  display: flex;
  gap: 2px;
}
</style>
