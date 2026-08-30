// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxt/fonts'
  ],

  // SPA mode - disable SSR for embedding in Go binary
  ssr: false,

  devtools: {
    enabled: true
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || ''
    }
  },

  compatibilityDate: '2025-01-15',

  // Output static files to backend for embedding.
  // publicDir is a subfolder of dist/ on purpose: Nitro wipes publicDir on
  // every build, which would destroy dist/.gitkeep (needed by go:embed when
  // the SPA hasn't been built yet). Keeping .gitkeep as a sibling of spa/
  // ensures `//go:embed all:dist` always finds at least one file.
  nitro: {
    output: {
      dir: '../backend/cmd/server/.output',
      publicDir: '../backend/cmd/server/dist/spa'
    }
  },

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  },

  // Font configuration
  fonts: {
    families: [
      {
        name: 'Manrope',
        provider: 'google',
        weights: [200, 300, 400, 500, 600, 700, 800]
      }
    ]
  }
})
