// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    'nuxt-oidc-auth'
  ],

  devtools: {
    enabled: true
  },

  css: ['~/assets/css/main.css'],

  routeRules: {
    '/': { prerender: false }
  },

  compatibilityDate: '2025-01-15',

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080'
    }
  },

  oidc: {
    enabled: true,
    defaultProvider: 'keycloak',
    providers: {
      keycloak: {
        clientId: process.env.NUXT_OIDC_CLIENT_ID || 'attic-web',
        clientSecret: '', // Public client, no secret needed
        redirectUri: process.env.NUXT_OIDC_REDIRECT_URI || 'http://localhost:3000/auth/keycloak/callback',
        baseUrl: process.env.NUXT_OIDC_ISSUER || 'http://localhost:8180/realms/attic',
        scope: ['openid', 'profile', 'email'],
        pkce: true,
        state: true,
        nonce: true,
        exposeAccessToken: true
      }
    },
    session: {
      automaticRefresh: true,
      expirationCheck: true
    },
    middleware: {
      globalMiddlewareEnabled: false,
      customLoginPage: false
    }
  },

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  }
})
