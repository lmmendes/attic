# RFC-008: Web and Mobile Authentication

| Field | Value |
|---|---|
| Status | In progress |
| Created | 2026-08-30 |

## Summary

Use Attic as the authentication boundary for browser and native clients. The
web client keeps an authenticated, encrypted `HttpOnly` cookie. Native clients
use OAuth 2.0 Authorization Code with PKCE and Attic-issued opaque bearer and
rotating refresh tokens.

Local passwords, OIDC, and future passkeys are login methods behind the same
Attic authorization endpoint. Native clients always use the system browser and
do not receive passwords or upstream identity-provider tokens.

## Native flow

1. The user enters the HTTPS URL of their Attic server.
2. The app reads `/auth/methods` and validates that all endpoints use the same
   origin.
3. The app opens `/oauth/authorize` in the system browser with a generated PKCE
   S256 challenge.
4. Attic authenticates the user locally or through its configured OIDC provider.
5. Attic redirects a short-lived, one-use code to the registered app URI.
6. The app exchanges the code and verifier at `/oauth/token`.
7. The app calls `/api/*` with the short-lived access token.
8. Refresh tokens rotate on every use. Reuse of an old token revokes its token
   family.

The built-in public client has no secret:

```text
client_id = attic-mobile
redirect_uri = com.lmmendes.attic:/oauth2redirect
```

## Web sessions

Browser code continues to use an `HttpOnly`, `Secure`, `SameSite=Lax` cookie.
Session payloads are authenticated and encrypted. Browser bearer tokens must
not be stored in local storage.

A later hardening change should move OIDC provider tokens out of the cookie and
into revocable server-side session storage.

## Tokens

- Authorization code lifetime: 2 minutes
- Access token lifetime: 15 minutes
- Refresh token lifetime: 30 days
- Token values: 256 bits from a cryptographically secure random source
- Database storage: SHA-256 hashes only
- Token responses: `Cache-Control: no-store`

## Passkeys

Passkeys will be implemented by the Attic authorization page using WebAuthn.
The relying-party ID is therefore the selected Attic server's web origin. The
platform authenticator retains the private key; neither the web application nor
Flutter receives it. Adding passkeys does not change the native OAuth protocol.

## Follow-up work

1. Persist the mobile refresh token in iOS Keychain and Android Keystore.
2. Add device-session listing and revocation to account settings.
3. Move web and upstream OIDC sessions to revocable server-side storage.
4. Add WebAuthn registration and authentication.
5. Prefer claimed HTTPS app links when a stable Attic-owned redirect domain is
   available; PKCE protects the current private-use URI callback.
