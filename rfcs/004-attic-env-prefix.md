# RFC-004: ATTIC_ Environment Variable Prefix

| Field       | Value                          |
|-------------|--------------------------------|
| Status      | Implemented                    |
| Created     | 2026-01-13                     |
| Author      | @lmmendes                      |

## Summary

Prefix all Attic-specific environment variables with `ATTIC_` to avoid naming collisions with other systems running in the same environment.

## Motivation

Environment variable names like `PORT`, `DATABASE_URL`, `BASE_URL`, and `SESSION_SECRET` are generic and commonly used by many applications. When running Attic alongside other services (e.g., in a shared Kubernetes namespace, Docker network, or development environment), these generic names can cause:

1. **Unintended collisions** - Another service may set `PORT=3000` which Attic inadvertently reads
2. **Configuration confusion** - Operators must carefully track which variables belong to which service
3. **Reduced debuggability** - Harder to grep logs and configs for Attic-specific settings
4. **CI/CD complexity** - Pipeline variables from different services may conflict

Adding the `ATTIC_` prefix follows industry best practices (e.g., `POSTGRES_*`, `KC_*`, `AWS_*`) and makes the configuration self-documenting.

## Scope

### Variables to Rename

The following Attic-specific variables will be prefixed with `ATTIC_`:

| Current Name       | New Name                  | Location                     |
|--------------------|---------------------------|------------------------------|
| `PORT`             | `ATTIC_PORT`              | backend/internal/config      |
| `DATABASE_URL`     | `ATTIC_DATABASE_URL`      | backend/internal/config      |
| `BASE_URL`         | `ATTIC_BASE_URL`          | backend/internal/config      |
| `CORS_ORIGINS`     | `ATTIC_CORS_ORIGINS`      | backend/internal/config      |
| `SESSION_SECRET`   | `ATTIC_SESSION_SECRET`    | backend/internal/config      |
| `AUTH_DISABLED`    | `ATTIC_AUTH_DISABLED`     | backend/internal/config      |
| `S3_ENDPOINT`      | `ATTIC_S3_ENDPOINT`       | backend/internal/config      |
| `S3_BUCKET`        | `ATTIC_S3_BUCKET`         | backend/internal/config      |
| `S3_REGION`        | `ATTIC_S3_REGION`         | backend/internal/config      |
| `S3_ACCESS_KEY`    | `ATTIC_S3_ACCESS_KEY`     | backend/internal/config      |
| `S3_SECRET_KEY`    | `ATTIC_S3_SECRET_KEY`     | backend/internal/config      |
| `OIDC_ISSUER`      | `ATTIC_OIDC_ISSUER_URL`   | backend/internal/config      |
| `OIDC_CLIENT_ID`   | `ATTIC_OIDC_CLIENT_ID`    | backend/internal/config      |
| `TMDB_API_KEY`     | `ATTIC_TMDB_API_KEY`      | backend/internal/plugin/tmdb |

### Variables NOT Renamed

The following variables will **not** be renamed as they are consumed by third-party services or follow established conventions:

| Variable                | Reason                                         |
|-------------------------|------------------------------------------------|
| `POSTGRES_USER`         | Standard PostgreSQL container variable         |
| `POSTGRES_PASSWORD`     | Standard PostgreSQL container variable         |
| `POSTGRES_DB`           | Standard PostgreSQL container variable         |
| `KEYCLOAK_ADMIN`        | Standard Keycloak container variable           |
| `KEYCLOAK_ADMIN_PASSWORD` | Standard Keycloak container variable         |
| `KC_*`                  | Standard Keycloak container variables          |
| `NUXT_PUBLIC_API_BASE`  | Follows Nuxt.js `NUXT_PUBLIC_*` convention     |
| `GITHUB_TOKEN`          | GitHub Actions standard variable               |

## Implementation

Since this is a greenfield project with no existing deployments, we will rename all variables directly without backward compatibility support.

### Step 1: Update Backend Config

Update `backend/internal/config/config.go` to use `ATTIC_` prefixed variable names in all `getEnv()` calls.

### Step 2: Update TMDB Plugin

Update `backend/internal/plugin/tmdb/common.go` to read `ATTIC_TMDB_API_KEY`.

### Step 3: Update Build Configuration

1. Update `Makefile` ldflags to use `ATTIC_TMDB_API_KEY`
2. Update `.goreleaser.yml` to use `ATTIC_TMDB_API_KEY`
3. Update GitHub Actions workflows

### Step 4: Update Docker Compose Files

1. Update `docker-compose.yml` to use `ATTIC_` prefix
2. Update `docker-compose.dev.yml` to use `ATTIC_` prefix

### Step 5: Update Documentation

1. Update `.env.example` with new variable names
2. Update `README.md` configuration section

## Files Affected

```
backend/internal/config/config.go      # Config loading logic
backend/internal/plugin/tmdb/common.go # TMDB API key loading
docker-compose.yml                     # Production compose file
docker-compose.dev.yml                 # Development compose file
.env.example                           # Example configuration
Makefile                               # Build automation
.goreleaser.yml                        # Release configuration
.github/workflows/release-please.yml   # CI/CD pipeline
frontend/nuxt.config.ts                # Frontend config (comment update)
README.md                              # Documentation
```

## Alternatives Considered

### 1. No Prefix
Keep variables as-is. Rejected because it doesn't solve the collision problem.

### 2. Shorter Prefix (e.g., `AT_`)
Use a 2-character prefix. Rejected because `ATTIC_` is more readable and self-documenting.

### 3. Application-specific Prefix Only for Conflicting Variables
Only prefix `PORT`, `DATABASE_URL`, etc. Rejected for inconsistency - partial prefixing is confusing.

## Open Questions

1. Should the frontend variable become `ATTIC_PUBLIC_API_BASE` or stay as `NUXT_PUBLIC_API_BASE`?
   - **Recommendation**: Keep `NUXT_PUBLIC_*` as it follows Nuxt.js conventions and is framework-specific.

## References

- [12-Factor App: Config](https://12factor.net/config)
- [Kubernetes ConfigMaps Best Practices](https://kubernetes.io/docs/concepts/configuration/configmap/)
