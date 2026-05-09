import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = (env.VITE_PROXY_TARGET || 'http://127.0.0.1:8080').replace(/\/$/, '')

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url))
      }
    },
    server: {
      host: '0.0.0.0',
      port: 5173,
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true
        },
        '/v1': {
          target: proxyTarget,
          changeOrigin: true
        },
        '/v1beta': {
          target: proxyTarget,
          changeOrigin: true
        },
        '/healthz': {
          target: proxyTarget,
          changeOrigin: true
        }
      }
    },
    build: {
      outDir: '../server/internal/web/dist',
      emptyOutDir: true
    }
  }
})
