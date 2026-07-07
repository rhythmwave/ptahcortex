# Same Codebase Comparison: Ptahcortex vs Claude Code

## Codebase: commit-reviewer (Go/Kotlin/TypeScript)

### Ptahcortex Findings (4 findings)

| # | Severity | File | Issue |
|---|----------|------|-------|
| 1 | High | `internal/auth/google.go:28` | OAuth2 URL construction |
| 2 | High | (needs verification) | Redirect URI validation |
| 3 | High | `internal/reviewer/engine.go:42-304` | Large method (263 lines) |
| 4 | Medium | `SettingsScreen.kt:78-199` | Large function (122 lines) |

### Claude Code Findings (10 findings)

| # | Severity | File | Issue |
|---|----------|------|-------|
| 1 | 🔴 Critical | `handler.go:84` | JWT in URL query parameter |
| 2 | 🔴 Critical | `config.go:54` | Weak default JWT secret |
| 3 | 🔴 Critical | `server.go:107-123` | CORS reflects any origin |
| 4 | 🟠 High | `session.go:22-76` | OAuth state no user binding |
| 5 | 🟠 High | `api.ts:104` | JWT in localStorage |
| 6 | 🟠 High | `service.go:146-158` | JWT no aud claim check |
| 7 | 🟠 High | `github.go:48-51` | JSON injection risk |
| 8 | 🟠 High | `store.go:123-135` | Tokens stored in plaintext |
| 9 | 🟡 Medium | `middleware.go:24` | Logs JWT token |
| 10 | 🟡 Medium | `handler.go:111-113` | No server-side revocation |

## Comparison

| Metric | Ptahcortex | Claude Code |
|--------|------------|-------------|
| **Critical** | 0 | 3 |
| **High** | 2 | 5 |
| **Medium** | 2 | 2 |
| **Total** | 4 | 10 |
| **Speed** | 🏆 28s | 56s |

## Why Claude Code Finds More

1. **More iterations** — 10+ vs 3
2. **More tool calls** — 10+ vs 3
3. **Reads more files** — handler.go, config.go, server.go, etc.
4. **Better context** — Startup folder structure

## Key Difference

**Ptahcortex:** Finds structural issues (large methods)
**Claude Code:** Finds security vulnerabilities (JWT, CORS, injection)

## Conclusion

**Claude Code is better for security audits** — Finds critical vulnerabilities.

**Ptahcortex is better for speed** — 28s vs 56s.

**Recommendation:** Use Claude Code for security-critical code, Ptahcortex for quick reviews.
