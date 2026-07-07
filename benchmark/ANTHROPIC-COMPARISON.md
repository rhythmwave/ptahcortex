# Anthropic Endpoint Comparison: Ptahcortex vs Claude Code

## Same Codebase: commit-reviewer

### Ptahcortex (Anthropic, 10 iterations, 50s)

| # | Severity | Issue |
|---|----------|-------|
| 1 | 🟠 High | Missing Middleware File |
| 2 | 🟠 High | JWT/Token Validation Gaps |
| 3 | 🟡 Medium | PKCE Implementation Risks |
| 4 | 🟡 Medium | Session Management Concerns |

### Claude Code (Anthropic, 10+ iterations, 56s)

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

| Metric | Ptahcortex (Anthropic) | Claude Code |
|--------|------------------------|-------------|
| **Critical** | 0 | 3 |
| **High** | 2 | 5 |
| **Medium** | 2 | 2 |
| **Total** | 4 | 10 |
| **Speed** | 🏆 50s | 56s |
| **Tokens** | ~5k | ~10k+ |

## Key Findings

### Both Use Anthropic Endpoint
- ✅ Same model (mimo-v2.5)
- ✅ Same endpoint
- ✅ Same API key

### Claude Code Finds More
- ❌ 10 findings vs 4 findings
- ❌ 3 Critical issues
- ❌ More detailed analysis

### Ptahcortex is Faster
- ✅ 50s vs 56s
- ✅ Fewer tokens
- ✅ More efficient

## Conclusion

**Both work with Anthropic endpoint.**

**Claude Code finds more issues** — 10 vs 4.

**Ptahcortex is faster** — 50s vs 56s.

**Same model, different results** — Claude Code's architecture is better for security audits.
