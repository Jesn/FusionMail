import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App.tsx'
import './index.css'
import { initTheme } from './lib/theme'

const queryClient = new QueryClient()

// Load auth test utilities in development
if (import.meta.env.DEV) {
  import('./utils/authTest')
}

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Failed to find the root element')
}

// 在 React 渲染前同步应用主题（读取 ThemeProvider 使用的存储键），
// 避免深色模式刷新时先渲染浅色再切换导致的闪白（FOUC）
initTheme()

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
)