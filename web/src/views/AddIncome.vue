<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const amount = ref('')
const error = ref('')
const busy = ref(false)
const result = ref(null)

const money = new Intl.NumberFormat(undefined, {
  style: 'currency',
  currency: 'USD',
})

function bucketName(id) {
  if (!result.value) return ''
  const b = result.value.buckets.find((x) => x.id === id)
  return b ? b.name : `#${id}`
}

async function submit() {
  error.value = ''
  const value = parseFloat(amount.value)
  if (!(value > 0)) {
    error.value = 'Enter an amount greater than 0'
    return
  }
  busy.value = true
  try {
    result.value = await api.post('/api/income', { amount: value })
    amount.value = ''
  } catch (e) {
    if (e.status === 401) return router.push('/login')
    error.value = e.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <RouterLink class="link" style="color: #fff" to="/">← Back</RouterLink>
      <h1>Add income</h1>
      <span style="width: 40px"></span>
    </header>

    <main class="content">
      <div v-if="error" class="error">{{ error }}</div>

      <form class="card" @submit.prevent="submit">
        <div class="field">
          <label for="amount">Amount received</label>
          <input
            id="amount"
            v-model="amount"
            type="number"
            inputmode="decimal"
            step="0.01"
            min="0"
            placeholder="0.00"
          />
        </div>
        <button class="btn" type="submit" :disabled="busy">
          {{ busy ? 'Distributing…' : 'Distribute to buckets' }}
        </button>
        <p class="muted" style="margin: 12px 0 0; font-size: 0.85rem">
          Your income is split across buckets by your distribution function.
          Buckets it doesn't reference are left untouched.
        </p>
      </form>

      <div v-if="result" class="card">
        <div class="title" style="font-size: 1.1rem; margin-bottom: 10px">
          Distributed {{ money.format(result.income.amount) }}
        </div>
        <div v-if="result.allocations && result.allocations.length">
          <div class="alloc-row" v-for="a in result.allocations" :key="a.bucket_id">
            <span>{{ bucketName(a.bucket_id) }}</span>
            <strong>{{ money.format(a.amount) }}</strong>
          </div>
        </div>
        <p v-else class="muted">No buckets were allocated to.</p>
        <RouterLink class="btn secondary" to="/" style="margin-top: 16px">View buckets</RouterLink>
      </div>
    </main>
  </div>
</template>
