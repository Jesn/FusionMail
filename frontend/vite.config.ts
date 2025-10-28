import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  optimizeDeps: {
    include: ['axios'],
  },
  server: {
    port: 3000,
    strictPort: true,
    host: true,
    proxy: {
      // 代理 API 请求到后端服务器
      '^/api/': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // 代理 WebSocket 请求
      '^/ws': {
        target: 'ws://localhost:8080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: true,
  },
})