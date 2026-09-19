import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 18528,
    proxy: {
      '/api': 'http://127.0.0.1:19528',
    },
  },
  test: {
    environment: 'jsdom',
  },
})
