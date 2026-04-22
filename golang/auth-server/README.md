# auth-server

A reusable, production-ready authentication server in Go.

## Features

| Feature | Details |
|---|---|
| Email + Password | Register, login, forgot/reset password, email verification |
| Social Login | Google, GitHub (add more providers in `service/social.go`) |
| OAuth 2.0 Server | Authorization Code + PKCE, Client Credentials, Refresh Token |
| Token Strategy | Short-lived JWT access tokens + rotating refresh tokens |
| Store Interface | In-memory (default) — swap for Postgres/Redis |
| Email Interface | Console (default) — swap for SMTP/SendGrid/SES |

## Quick Start

```bash
cp .env.example .env
# edit .env — set JWT_SECRET at minimum

go mod tidy
make run
```

## Folder Structure

```
auth-server/
├── cmd/server/main.go          # Entrypoint — wires everything
├── internal/
│   ├── config/config.go        # Env-based config
│   ├── model/                  # Domain types (User, Token, OAuth)
│   ├── store/
│   │   ├── interfaces.go       # UserStore, TokenStore, OAuthStore interfaces
│   │   └── memory/memory.go    # In-memory implementation (replace for prod)
│   ├── service/
│   │   ├── auth.go             # Email/password, forgot/reset, verify
│   │   ├── social.go           # Google & GitHub social login
│   │   └── oauth.go            # OAuth 2.0 authorization server
│   ├── handler/
│   │   ├── auth.go             # HTTP handlers for auth endpoints
│   │   ├── social.go           # HTTP handlers for social login
│   │   ├── oauth.go            # HTTP handlers for OAuth endpoints
│   │   └── response.go         # Shared JSON helpers
│   ├── middleware/auth.go      # JWT middleware (Required, Optional, RequireScope)
│   └── router/router.go        # Chi router — all routes wired here
└── pkg/
    ├── jwtutil/jwtutil.go      # JWT sign/parse (HS256, swap to RS256 easily)
    └── password/password.go    # bcrypt helpers
```

## API Endpoints

### Email / Password

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | — | Create account |
| POST | `/auth/login` | — | Login, returns token pair |
| POST | `/auth/refresh` | — | Rotate refresh token |
| POST | `/auth/logout` | — | Revoke refresh token |
| POST | `/auth/logout-all` | Bearer | Revoke all sessions |
| POST | `/auth/forgot-password` | — | Send reset email |
| POST | `/auth/reset-password` | — | Set new password with token |
| GET | `/auth/verify-email?token=` | — | Verify email address |
| POST | `/auth/resend-verification` | Bearer | Resend verification email |

### Social Login

| Method | Path | Description |
|--------|------|-------------|
| GET | `/auth/social/google` | Redirect to Google consent |
| GET | `/auth/social/google/callback` | Google OAuth callback |
| GET | `/auth/social/github` | Redirect to GitHub consent |
| GET | `/auth/social/github/callback` | GitHub OAuth callback |

### OAuth 2.0 Server

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/oauth/authorize` | Bearer | Issue authorization code |
| POST | `/oauth/token` | Client creds | Exchange code / refresh token |
| POST | `/oauth/revoke` | — | Revoke token (RFC 7009) |
| POST | `/oauth/introspect` | — | Inspect token (RFC 7662) |
| GET | `/oauth/userinfo` | Bearer | OIDC userinfo |
| GET | `/.well-known/oauth-authorization-server` | — | Discovery metadata (RFC 8414) |

## Swapping the Store (Postgres example)

Implement the three interfaces in `internal/store/interfaces.go`:

```go
type UserStore interface { ... }
type TokenStore interface { ... }
type OAuthStore interface { ... }
```

Then in `cmd/server/main.go`, replace:
```go
st := memory.New()
```
with your implementation:
```go
st := postgres.New(db)   // your package
```

## Adding a Social Provider (e.g. Apple)

1. Add `ProviderApple AuthProvider = "apple"` to `model/user.go`.
2. In `service/social.go`, add an Apple `SocialProvider` entry in `NewSocialService`.
3. Implement `fetchAppleProfile` following the Google/GitHub pattern.
4. Add the redirect URL to your Apple app config.
