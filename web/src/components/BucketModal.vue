<script setup>
import { ref, computed } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  bucketId: { type: Number, required: true },
  buckets: { type: Array, required: true },
})
const emit = defineEmits(['close', 'changed', 'deleted'])

const money = new Intl.NumberFormat(undefined, { style: 'currency', currency: 'USD' })

const bucket = computed(() => props.buckets.find((b) => b.id === props.bucketId) || null)
const others = computed(() => props.buckets.filter((b) => b.id !== props.bucketId))

const error = ref('')
const notice = ref('')
const busy = ref(false)

// Move money
const moveAmount = ref('')
const moveTo = ref(others.value.length ? others.value[0].id : null)

async function move() {
  error.value = ''
  notice.value = ''
  const amount = parseFloat(moveAmount.value)
  if (!(amount > 0)) {
    error.value = 'Enter an amount greater than 0'
    return
  }
  if (!moveTo.value) {
    error.value = 'Pick a destination bucket'
    return
  }
  busy.value = true
  try {
    const updated = await api.post(`/api/buckets/${props.bucketId}/transfer`, {
      to_bucket_id: moveTo.value,
      amount,
    })
    emit('changed', updated)
    const dest = updated.find((b) => b.id === moveTo.value)
    notice.value = `Moved ${money.format(amount)} to ${dest ? dest.name : 'bucket'}`
    moveAmount.value = ''
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

// Edit
const editing = ref(false)
const editName = ref('')
const editDesc = ref('')
const editAmount = ref('')

function startEdit() {
  editName.value = bucket.value.name
  editDesc.value = bucket.value.description
  editAmount.value = String(bucket.value.current_amount)
  editing.value = true
  error.value = ''
  notice.value = ''
}

async function saveEdit() {
  error.value = ''
  if (!editName.value.trim()) {
    error.value = 'Name is required'
    return
  }
  busy.value = true
  try {
    await api.put(`/api/buckets/${props.bucketId}`, {
      name: editName.value.trim(),
      description: editDesc.value.trim(),
      current_amount: parseFloat(editAmount.value) || 0,
    })
    const list = await api.get('/api/buckets')
    emit('changed', list)
    editing.value = false
    notice.value = 'Saved'
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (!confirm(`Delete "${bucket.value.name}"? This cannot be undone.`)) return
  busy.value = true
  try {
    await api.del(`/api/buckets/${props.bucketId}`)
    emit('deleted', props.bucketId)
  } catch (e) {
    error.value = e.message
    busy.value = false
  }
}
</script>

<template>
  <div class="sheet-backdrop" @click.self="emit('close')">
    <div class="sheet" role="dialog" aria-modal="true">
      <div class="sheet-grip"></div>

      <div v-if="bucket" class="sheet-head">
        <div>
          <div class="sheet-title">{{ bucket.name }}</div>
          <div class="sheet-balance">{{ money.format(bucket.current_amount) }}</div>
        </div>
        <button class="icon-btn" aria-label="Close" @click="emit('close')">✕</button>
      </div>

      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="notice" class="notice">{{ notice }}</div>

      <template v-if="bucket && !editing">
        <!-- Move money -->
        <div class="section">
          <div class="section-title">Move money</div>
          <div v-if="others.length === 0" class="muted">Add another bucket to move money.</div>
          <template v-else>
            <div class="field">
              <label>Amount</label>
              <input v-model="moveAmount" type="number" inputmode="decimal" step="0.01" min="0" placeholder="0.00" />
            </div>
            <div class="field">
              <label>To bucket</label>
              <select v-model="moveTo" class="select">
                <option v-for="b in others" :key="b.id" :value="b.id">{{ b.name }}</option>
              </select>
            </div>
            <button class="btn" :disabled="busy" @click="move">Move money</button>
          </template>
        </div>

        <div class="row-actions">
          <button class="btn secondary" :disabled="busy" @click="startEdit">Edit</button>
          <button class="btn danger" :disabled="busy" @click="remove">Delete</button>
        </div>
      </template>

      <!-- Edit mode -->
      <template v-else-if="bucket && editing">
        <div class="section">
          <div class="section-title">Edit bucket</div>
          <div class="field">
            <label>Name</label>
            <input v-model="editName" type="text" />
          </div>
          <div class="field">
            <label>Description</label>
            <input v-model="editDesc" type="text" />
          </div>
          <div class="field">
            <label>Balance</label>
            <input v-model="editAmount" type="number" inputmode="decimal" step="0.01" />
          </div>
          <button class="btn" :disabled="busy" @click="saveEdit">Save changes</button>
          <button class="btn ghost" :disabled="busy" @click="editing = false">Cancel</button>
        </div>
      </template>
    </div>
  </div>
</template>
