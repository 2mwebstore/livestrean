import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: false },
  modules: ['@pinia/nuxt'],
  css: ['~/assets/css/main.css'],
  vite: { plugins: [tailwindcss()] },

  // Feature components are auto-imported by file name (ChatPanel, LivePlayer...).
  // shadcn-vue components in components/ui are imported explicitly.
  components: [{ path: '~/components', pathPrefix: false, ignore: ['ui/**'] }],

  runtimeConfig: {
    // Server-only: during SSR, call the Go API directly inside the Docker
    // network. Override with NUXT_API_INTERNAL_BASE.
    apiInternalBase: 'http://localhost:8080/api/v1',
    public: {
      // Browser: same origin, routed to the Go API by Caddy.
      apiBase: '/api/v1',
      sessionCookie: 'ls_session',
      siteName: 'Livestream',
    },
  },

  // Public pages render on the server (SEO, fast first paint). The dashboard
  // and admin are private, interactive apps rendered in the browser.
  routeRules: {
    '/dashboard': { ssr: false },
    '/dashboard/**': { ssr: false },
    '/admin': { ssr: false },
    '/admin/**': { ssr: false },
  },

  app: {
    head: {
      htmlAttrs: { lang: 'en', class: 'dark' },
      link: [{ rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    },
  },

  // `npm run dev` outside Docker: proxy API, WebSocket and video to the stack.
  nitro: {
    devProxy: {
      '/api': { target: 'http://localhost:8080/api', changeOrigin: false },
      '/ws': { target: 'http://localhost:8080/ws', ws: true },
      '/live': { target: 'http://localhost:8080/live' },
    },
  },

  typescript: { strict: true },
})
