import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      registerType: 'autoUpdate',
      includeAssets: ['agent-message-logo.svg', 'favicon-0.6.21.png', 'apple-touch-icon-0.6.21.png'],
      manifest: {
        name: 'Agent Message',
        short_name: 'Agent Message',
        description: 'Phone-first direct messaging app for Agent Message.',
        theme_color: '#1f2228',
        background_color: '#1f2228',
        display: 'standalone',
        orientation: 'portrait',
        start_url: '/',
        scope: '/',
        icons: [
          {
            src: '/pwa-192x192-0.6.21.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: '/pwa-512x512-0.6.21.png',
            sizes: '512x512',
            type: 'image/png',
          },
          {
            src: '/pwa-512x512-0.6.21.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },
      injectManifest: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,webmanifest}'],
      },
    }),
  ],
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/static/uploads': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
