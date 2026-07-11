<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { login } from '../store.js'

const router = useRouter()
const email = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await login(email.value, password.value)
    router.push('/')
  } catch (e) {
    error.value = e.message || 'Login failed'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="center-screen">
    <div class="brand">
      <h1>Budget</h1>
      <p>Sign in to your buckets</p>
    </div>

    <form class="card" @submit.prevent="submit">
      <div v-if="error" class="error">{{ error }}</div>
      <div class="field">
        <label for="email">Email</label>
        <input id="email" v-model="email" type="email" autocomplete="email" required />
      </div>
      <div class="field">
        <label for="password">Password</label>
        <input id="password" v-model="password" type="password" autocomplete="current-password" required />
      </div>
      <button class="btn" type="submit" :disabled="busy">
        {{ busy ? 'Signing in…' : 'Sign in' }}
      </button>
    </form>

    <p class="muted" style="text-align: center; margin-top: 16px">
      No account?
      <button class="link" @click="router.push('/register')">Create one</button>
    </p>
  </div>
</template>
