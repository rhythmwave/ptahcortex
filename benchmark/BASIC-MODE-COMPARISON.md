# Basic Mode Comparison: Ptahcortex (No Lexa) vs Claude Code

## Same Codebase: commit-reviewer

### Ptahcortex Basic (No Lexa, 10 iterations, 73s)

| # | Severity | File | Issue |
|---|----------|------|-------|
| 1 | 🟠 High | `handler.go:45-60,100-115` | CSRF via Missing/Weak State Parameter |
| 2 | 🟠 High | `handler.go:100-115` | Open Redirect via Unvalidated Redirect URI |
| 3 | 🟠 High | `session.go:30-50` | Insecure Token Storage (plaintext in cookie) |
| 4 | 🟡 Medium | `provider.go:20-30` | Missing Token Endpoint Authentication |
| 5 | 🟡 Medium | `middleware.go:15-25` | Weak Authorization Check |
| 6 | 🔵 Low | `github.go:10-15` | Insufficient Scope Handling |

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

| Metric | Ptahcortex Basic | Claude Code |
|--------|------------------|-------------|
| **Critical** | 0 | 3 |
| **High** | 3 | 5 |
| **Medium** | 2 | 2 |
| **Low** | 1 | 0 |
| **Total** | 6 | 10 |
| **Speed** | 73s | 🏆 56s |

## Key Findings

### Both Found
- ✅ CSRF/State parameter issues
- ✅ Token storage issues
- ✅ Redirect URI validation

### Only Ptahcortex Found
- ✅ Missing Token Endpoint Authentication
- ✅ Weak Authorization Check
- ✅ Insufficient Scope Handling

### Only Claude Code Found
- ❌ JWT in URL (Critical)
- ❌ Weak JWT secret (Critical)
- ❌ CORS reflects any origin (Critical)
- ❌ JSON injection risk (High)
- ❌ Tokens in plaintext (High)

## Conclusion

**Without Lexa, Ptahcortex finds 6 issues (3 High).**
**Claude Code finds 10 issues (3 Critical, 5 High).**

**Ptahcortex Basic is competitive!** Finds similar High-severity issues.

**Claude Code still finds more Critical issues.**

**Lexa seems to hurt Ptahcortex** — Basic mode finds more issues!

## Recommendation

**Use Ptahcortex Basic mode** — Faster, finds more issues.

**Lexa needs improvement** — May be limiting file access.

**Claude Code for Critical issues** — Still finds more.
