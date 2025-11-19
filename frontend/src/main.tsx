import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './App.tsx'
import './index.css'
import { initTheme } from './lib/theme'

const queryClient = new QueryClient()

// Apply theme before rendering to reduce FOUC
initTheme()

// Load auth test utilities in development
if (import.meta.env.DEV) {
  import('./utils/authTest')
}

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Failed to find the root element')
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
)