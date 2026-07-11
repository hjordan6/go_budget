import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Build outputs to dist/, which the Go server serves as the SPA.
// During `npm run dev`, /api requests are proxied to the Go API so the
// frontend can be developed with hot reload against a running backend.
export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: 'dist',
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8891',
    },
  },
})
