import { createRouter, createWebHistory } from 'vue-router'
import { auth, ensureLoaded } from './store.js'

import BucketList from './views/BucketList.vue'
import AddIncome from './views/AddIncome.vue'
import AddBucket from './views/AddBucket.vue'
import Transactions from './views/Transactions.vue'
import LunchMoney from './views/LunchMoney.vue'
import Settings from './views/Settings.vue'
import Login from './views/Login.vue'
import Register from './views/Register.vue'

const routes = [
  { path: '/', component: BucketList, meta: { requiresAuth: true } },
  { path: '/income', component: AddIncome, meta: { requiresAuth: true } },
  { path: '/bucket/new', component: AddBucket, meta: { requiresAuth: true } },
  { path: '/transactions', component: Transactions, meta: { requiresAuth: true } },
  { path: '/lunchmoney', component: LunchMoney, meta: { requiresAuth: true } },
  { path: '/settings', component: Settings, meta: { requiresAuth: true } },
  { path: '/login', component: Login },
  { path: '/register', component: Register },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  await ensureLoaded()
  if (to.meta.requiresAuth && !auth.user) {
    return { path: '/login' }
  }
  if ((to.path === '/login' || to.path === '/register') && auth.user) {
    return { path: '/' }
  }
})

export default router
