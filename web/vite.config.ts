import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import ui from '@nuxt/ui/vite'

export default defineConfig({
  plugins: [vue(), ui()],
  server: {
    proxy: {
      '^/nostra\..*': {
        target: 'http://localhost:8080',
        changeOrigin: false,
      },
    },
  },
})
