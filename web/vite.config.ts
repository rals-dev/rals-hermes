import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

// In development the dev server proxies /api to a BFF so cookies stay
// same-origin, exactly as in production behind Traefik. Default is a local
// `make run` on :8080; set VITE_API_TARGET to point at a deployed instance.
const apiTarget = process.env.VITE_API_TARGET ?? 'http://127.0.0.1:8080'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      '/api': { target: apiTarget, changeOrigin: true },
    },
  },
  build: {
    outDir: 'dist',
    // dist/.gitkeep must survive so `go:embed all:dist` compiles without a
    // frontend build; the build script clears dist/assets itself.
    emptyOutDir: false,
    sourcemap: false,
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
})
