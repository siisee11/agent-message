import React from 'react'
import ReactDOM from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import { registerSW } from 'virtual:pwa-register'
import '@fontsource/geist/400.css'
import '@fontsource/geist/500.css'
import '@fontsource/geist-mono/300.css'
import '@fontsource/geist-mono/400.css'
import '@fontsource/geist-mono/500.css'
import { App } from './App'
import { AuthProvider } from './auth'
import './styles/global.css'
import { ThemeProvider } from './theme'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

let reloadedForServiceWorker = false
if ('serviceWorker' in navigator) {
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (reloadedForServiceWorker) {
      return
    }
    reloadedForServiceWorker = true
    window.location.reload()
  })
}

registerSW({
  immediate: true,
})

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <BrowserRouter>
            <App />
          </BrowserRouter>
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  </React.StrictMode>,
)
