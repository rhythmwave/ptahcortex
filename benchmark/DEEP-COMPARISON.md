# Deep Comparison: Ptahcortex (10 iterations) vs Claude Code

## Same Codebase: commit-reviewer

### Ptahcortex (10 iterations, 92s)

| # | Severity | File | Issue |
|---|----------|------|-------|
| 1 | 🟠 High | `internal/auth/handler.go:85-86` | JWT token in redirect URL |
| 2 | 🟡 Medium | `internal/auth/google.go:28` | Hardcoded redirect URI |
| 3 | 🟡 Medium | `internal/auth/github.go:36,49` | Dynamic redirect URI risk |
| 4 | ℹ️ Info | (needs verification) | CSRF state parameter |
| 5 | ℹ️ Info | (needs verification) | Token storage (localStorage) |

### Claude Code (10+ iterations, 56s)

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

| Metric | Ptahcortex (10 iter) | Claude Code |
|--------|----------------------|-------------|
| **Critical** | 0 | 3 |
| **High** | 1 | 5 |
| **Medium** | 2 | 2 |
| **Total** | 5 | 10 |
| **Speed** | 92s | 🏆 56s |
| **Iterations** | 10 | 10+ |

## Key Findings

### Both Found
- ✅ JWT token in URL (High/Critical)
- ✅ Token storage issues
- ✅ CSRF concerns

### Only Claude Code Found
- ❌ Weak default JWT secret (Critical)
- ❌ CORS reflects any origin (Critical)
- ❌ OAuth state no user binding (High)
- ❌ JSON injection risk (High)
- ❌ Tokens stored in plaintext (High)

## Why Claude Code Still Finds More

1. **Reads more files** — handler.go, config.go, server.go, etc.
2. **Better tool usage** — Uses multiple tools per iteration
3. **Startup context** — Folder structure upfront
4. **More comprehensive** — Checks JWT secret, CORS, etc.

## Conclusion

**With 10 iterations, Ptahcortex finds 5 issues (1 High).**
**Claude Code finds 10 issues (3 Critical, 5 High).**

**Claude Code still better for security audits** — Finds critical issues Ptahcortex misses.

**Ptahcortex faster** — 92s vs 56s (but more iterations = slower).

**Recommendation:** Use Claude Code for security-critical code.
