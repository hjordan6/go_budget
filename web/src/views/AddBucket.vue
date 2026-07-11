<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api.js'

const router = useRouter()
const name = ref('')
const description = ref('')
const amount = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  if (!name.value.trim()) {
    error.value = 'Name is required'
    return
  }
  busy.value = true
  try {
    await api.post('/api/buckets', {
      name: name.value.trim(),
      description: description.value.trim(),
      current_amount: parseFloat(amount.value) || 0,
    })
    router.push('/')
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
      <h1>New bucket</h1>
      <span style="width: 40px"></span>
    </header>

    <main class="content">
      <div v-if="error" class="error">{{ error }}</div>

      <form class="card" @submit.prevent="submit">
        <div class="field">
          <label for="name">Name</label>
          <input id="name" v-model="name" type="text" placeholder="e.g. Groceries" />
        </div>
        <div class="field">
          <label for="description">Description (optional)</label>
          <input id="description" v-model="description" type="text" />
        </div>
        <div class="field">
          <label for="amount">Starting amount (optional)</label>
          <input id="amount" v-model="amount" type="number" inputmode="decimal" step="0.01" placeholder="0.00" />
        </div>
        <button class="btn" type="submit" :disabled="busy">
          {{ busy ? 'Saving…' : 'Add bucket' }}
        </button>
        <p class="muted" style="margin: 12px 0 0; font-size: 0.85rem">
          New buckets are ignored by your distribution function until it's
          updated to include them.
        </p>
      </form>
    </main>
  </div>
</template>
