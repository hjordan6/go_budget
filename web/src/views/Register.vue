<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { register } from '../store.js'

const router = useRouter()
const email = ref('')
const name = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await register(email.value, name.value, password.value)
    router.push('/')
  } catch (e) {
    error.value = e.message || 'Registration failed'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="center-screen">
    <div class="brand">
      <h1>Budget</h1>
      <p>Create your account</p>
    </div>

    <form class="card" @submit.prevent="submit">
      <div v-if="error" class="error">{{ error }}</div>
      <div class="field">
        <label for="name">Name</label>
        <input id="name" v-model="name" type="text" autocomplete="name" />
      </div>
      <div class="field">
        <label for="email">Email</label>
        <input id="email" v-model="email" type="email" autocomplete="email" required />
      </div>
      <div class="field">
        <label for="password">Password</label>
        <input id="password" v-model="password" type="password" autocomplete="new-password" required />
      </div>
      <button class="btn" type="submit" :disabled="busy">
        {{ busy ? 'Creating…' : 'Create account' }}
      </button>
    </form>

    <p class="muted" style="text-align: center; margin-top: 16px">
      Already have an account?
      <button class="link" @click="router.push('/login')">Sign in</button>
    </p>
  </div>
</template>
