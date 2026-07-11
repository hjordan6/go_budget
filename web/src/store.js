import { reactive } from 'vue'
import { api } from './api.js'

// Global auth state. `loaded` tracks whether we've checked the session yet so
// the router guard can wait for the initial /api/me before deciding routes.
export const auth = reactive({
  user: null,
  loaded: false,
})

// ensureLoaded checks the current session once on app start.
export async function ensureLoaded() {
  if (auth.loaded) return
  try {
    auth.user = await api.get('/api/me')
  } catch {
    auth.user = null
  } finally {
    auth.loaded = true
  }
}

export async function login(email, password) {
  auth.user = await api.post('/api/login', { email, password })
}

export async function register(email, name, password) {
  auth.user = await api.post('/api/register', { email, name, password })
}

export async function logout() {
  try {
    await api.post('/api/logout')
  } finally {
    auth.user = null
  }
}
