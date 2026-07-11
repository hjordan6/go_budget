<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import draggable from 'vuedraggable'
import { api } from '../api.js'
import { auth, logout } from '../store.js'
import BucketModal from '../components/BucketModal.vue'

const router = useRouter()
const buckets = ref([])
const error = ref('')
const loading = ref(true)
const selectedId = ref(null)

const money = new Intl.NumberFormat(undefined, {
  style: 'currency',
  currency: 'USD',
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    buckets.value = await api.get('/api/buckets')
  } catch (e) {
    if (e.status === 401) return router.push('/login')
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function onReorder() {
  try {
    const ids = buckets.value.map((b) => b.id)
    buckets.value = await api.put('/api/buckets/reorder', { ids })
  } catch (e) {
    error.value = e.message
    load() // resync on failure
  }
}

async function doLogout() {
  await logout()
  router.push('/login')
}

function onModalChanged(updated) {
  buckets.value = updated
}

function onModalDeleted(id) {
  buckets.value = buckets.value.filter((b) => b.id !== id)
  selectedId.value = null
}

onMounted(load)
</script>

<template>
  <div class="app-shell">
    <header class="app-header">
      <div>
        <h1>Budget</h1>
        <div class="user" v-if="auth.user">{{ auth.user.name || auth.user.email }}</div>
      </div>
      <div style="display: flex; flex-direction: column; align-items: flex-end; gap: 4px">
        <RouterLink class="link" style="color: #fff" to="/transactions">Transactions</RouterLink>
        <button class="link" style="color: #fff" @click="doLogout">Log out</button>
      </div>
    </header>

    <main class="content">
      <div v-if="error" class="error">{{ error }}</div>

      <p v-if="loading" class="muted">Loading…</p>

      <div v-else-if="buckets.length === 0" class="empty">
        No buckets yet.<br />Add one to get started.
      </div>

      <draggable
        v-else
        v-model="buckets"
        item-key="id"
        handle=".handle"
        ghost-class="ghost"
        animation="150"
        @end="onReorder"
      >
        <template #item="{ element }">
          <div class="card bucket-card">
            <div class="handle" aria-label="Drag to reorder">⠿</div>
            <button class="bucket-main" type="button" @click="selectedId = element.id">
              <div class="info">
                <div class="name">{{ element.name }}</div>
                <div v-if="element.description" class="desc">{{ element.description }}</div>
              </div>
              <div class="amount">{{ money.format(element.current_amount) }}</div>
            </button>
          </div>
        </template>
      </draggable>
    </main>

    <nav class="actions-bar" v-if="selectedId === null">
      <RouterLink class="btn secondary" to="/bucket/new">+ Bucket</RouterLink>
      <RouterLink class="btn" to="/income">+ Add income</RouterLink>
    </nav>

    <BucketModal
      v-if="selectedId !== null"
      :bucket-id="selectedId"
      :buckets="buckets"
      @close="selectedId = null"
      @changed="onModalChanged"
      @deleted="onModalDeleted"
    />
  </div>
</template>
