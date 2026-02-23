# RFC-005: Email and Password Authentication

| Field       | Value                          |
|-------------|--------------------------------|
| Status      | Implemented                    |
| Created     | 2026-01-13                     |
| Author      | @lmmendes                       |

## Summary

Add email/password authentication as the default authentication method for Attic, with OIDC as an optional feature. Implement role-based access control (RBAC) with two roles: `user` and `admin`.

## Motivation

Currently, Attic only supports OIDC authentication via Keycloak, which requires additional infrastructure setup. Many deployments would benefit from a simpler built-in authentication system that works out of the box.

Email/password authentication provides:
1. **Zero-config startup** - Works immediately without external identity provider
2. **Simpler deployments** - No need to run Keycloak for small installations
3. **Self-contained** - All user data stored in the existing PostgreSQL database

## User Model

### Fields

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Primary key |
| email | string | Unique identifier (not validated as email format), lowercase |
| name | string | Display name |
| password_hash | string | bcrypt hashed password |
| role | enum | `user` or `admin` |
| oidc_subject | string | OIDC subject claim (for linked accounts) |
| created_at | timestamp | Account creation time |
| updated_at | timestamp | Last modification time |

### Roles

| Role | Permissions |
|------|-------------|
| `user` | Login, view assets, change own password |
| `admin` | All user permissions + manage users (create, delete, change roles, reset passwords, change emails) |

## Authentication Modes

### Mode 1: Email/Password (Default)

- Enabled by default when `ATTIC_OIDC_ENABLED` is not set or `false`
- Users authenticate with email and password
- Sessions last 24 hours (configurable via `ATTIC_SESSION_DURATION_HOURS`)

### Mode 2: OIDC

- Enabled when `ATTIC_OIDC_ENABLED=true`
- Email/password login is disabled
- Users authenticate via configured OIDC provider
- Admin UI for user management remains accessible

### Account Linking

When OIDC is enabled and a user logs in:
1. If a user with matching email exists, link the OIDC subject to that account
2. The user inherits the existing role (preserves admin status)
3. If no matching user exists, create new user with `user` role

When OIDC is disabled after being enabled:
1. Users with email/password can log in again
2. Users created via OIDC only (no password set) cannot log in until admin resets their password

## Bootstrap Admin

On first startup, if no users exist in the database:
1. Create an admin user with:
   - Email: value of `ATTIC_ADMIN_EMAIL` or `"admin"`
   - Password: value of `ATTIC_ADMIN_PASSWORD` or `"admin"`
   - Role: `admin`
2. Log a warning if using default credentials

## Password Requirements

- Minimum length: configurable via `ATTIC_PASSWORD_MIN_LENGTH` (default: 8)
- No complexity requirements enforced
- Passwords hashed with bcrypt

## Password Reset

Admins can reset any user's password via:
1. **Admin UI** - User management section
2. **CLI** - For recovery when locked out:

```bash
docker exec -it <container> ./attic --reset-password --email="admin" --new-password="NewPassword123"
```

The CLI command:
- Works regardless of authentication mode
- Does not require authentication (for recovery scenarios)
- Exits with error if user not found

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ATTIC_OIDC_ENABLED` | Enable OIDC authentication | `false` |
| `ATTIC_ADMIN_EMAIL` | Bootstrap admin email | `admin` |
| `ATTIC_ADMIN_PASSWORD` | Bootstrap admin password | `admin` |
| `ATTIC_SESSION_DURATION_HOURS` | Session lifetime in hours | `24` |
| `ATTIC_PASSWORD_MIN_LENGTH` | Minimum password length | `8` |

Existing OIDC variables (only used when `ATTIC_OIDC_ENABLED=true`):
- `ATTIC_OIDC_ISSUER_URL`
- `ATTIC_OIDC_CLIENT_ID`

## API Endpoints

### Authentication

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/auth/login` | Email/password login | No |
| POST | `/api/auth/logout` | End session | Yes |
| GET | `/api/auth/me` | Current user info | Yes |
| PUT | `/api/auth/password` | Change own password | Yes |

### User Management (Admin only)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/users` | List all users |
| POST | `/api/users` | Create new user |
| GET | `/api/users/:id` | Get user details |
| PUT | `/api/users/:id` | Update user (email, name, role) |
| DELETE | `/api/users/:id` | Delete user |
| POST | `/api/users/:id/reset-password` | Reset user's password |

## Frontend Changes

### Login Page

- Show email/password form when OIDC disabled
- Show "Login with SSO" button when OIDC enabled
- Display appropriate error messages

### User Management Page (Admin only)

New page at `/users` (access restricted to admin role) with:
- User list table (email, name, role, created date)
- Create user modal
- Edit user modal (change email, name, role)
- Delete user confirmation
- Reset password action

### Profile/Settings

- Allow all users to change their own password
- Show "Change Password" only when OIDC disabled

## Database Migration

```sql
CREATE TYPE user_role AS ENUM ('user', 'admin');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL DEFAULT '',
    password_hash VARCHAR(255),
    role user_role NOT NULL DEFAULT 'user',
    oidc_subject VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_oidc_subject ON users(oidc_subject);
```

## Implementation Steps

### Backend

1. Add database migration for users table
2. Implement user repository (CRUD operations)
3. Implement password hashing service (bcrypt)
4. Implement session management (JWT or server-side sessions)
5. Add authentication middleware (check mode, validate session)
6. Add authorization middleware (check role for admin endpoints)
7. Implement auth endpoints (login, logout, change password)
8. Implement user management endpoints
9. Add CLI command for password reset
10. Update bootstrap logic to create initial admin
11. Update OIDC flow to link/create users

### Frontend

1. Create login page with email/password form
2. Update auth composable for new auth flow
3. Create user management page
4. Add password change to settings/profile
5. Update navigation to show admin menu items
6. Handle auth mode switching (OIDC vs email/password)

## Security Considerations

1. **Password Storage**: bcrypt with default cost factor (10)
2. **Session Tokens**: Cryptographically secure random tokens
3. **HTTPS**: Required in production for credential transmission
4. **Rate Limiting**: Not implemented initially (out of scope)
5. **Account Lockout**: Not implemented (out of scope)

## Files Affected

```
backend/
├── cmd/server/main.go              # CLI password reset, bootstrap admin
├── internal/
│   ├── auth/
│   │   ├── middleware.go           # Update for dual auth modes
│   │   ├── password.go             # New: bcrypt utilities
│   │   └── session.go              # New: session management
│   ├── config/config.go            # New env variables
│   ├── domain/user.go              # New: user model
│   ├── handler/
│   │   ├── auth.go                 # New: auth endpoints
│   │   └── user.go                 # New: user management
│   └── repository/user.go          # New: user repository
├── migrations/
│   └── XXXXXX_add_users.up.sql     # New migration

frontend/
├── app/
│   ├── pages/
│   │   ├── login.vue               # New/update login page
│   │   └── users.vue               # New: user management (admin only)
│   ├── components/
│   │   └── users/
│   │       ├── UserTable.vue       # New
│   │       ├── UserModal.vue       # New
│   │       └── ResetPasswordModal.vue  # New
│   └── composables/
│       └── useAuth.ts              # Update for new auth
```

## Open Questions

None - all questions resolved during RFC discussion.

## References

- RFC-004: ATTIC_ Environment Variable Prefix
- [bcrypt](https://en.wikipedia.org/wiki/Bcrypt)
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
