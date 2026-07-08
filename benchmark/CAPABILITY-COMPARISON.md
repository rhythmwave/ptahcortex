# Capability Comparison: Ptahcortex vs Claude Code

## Tool Comparison

### Read Operations

| Capability | Claude Code | Ptahcortex | Status |
|------------|-------------|------------|--------|
| Read file | ✅ `read` | ✅ `read_file` + `read` (Lexa) | ✅ |
| List files | ✅ `list_files` | ✅ `list_files` + `files` (Lexa) | ✅ |
| Search code | ✅ `grep` | ✅ `text_search` (Lexa) | ✅ |
| File structure | ✅ `outline` | ✅ `outline` (Lexa) | ✅ |
| Symbol search | ✅ `search` | ✅ `symbol_search` (Lexa) | ✅ |
| Find references | ✅ `references` | ✅ `word_refs` (Lexa) | ✅ |
| Call graph | ❌ | ✅ `callers` (Lexa) | 🏆 |
| Dependency trace | ❌ | ✅ `trace_deps` (Lexa) | 🏆 |
| Glob patterns | ✅ `glob` | ✅ `glob` (Lexa) | ✅ |

### Write Operations

| Capability | Claude Code | Ptahcortex | Status |
|------------|-------------|------------|--------|
| Write file | ✅ `write` | ✅ `write_file` | ✅ |
| Edit file | ✅ `edit` | ✅ `patch` (Lexa) | ✅ |
| Create file | ✅ `write` | ✅ `create` (Lexa) | ✅ |
| Apply diff | ✅ `edit` | ✅ `patch` (Lexa) | ✅ |

### Execute Operations

| Capability | Claude Code | Ptahcortex | Status |
|------------|-------------|------------|--------|
| Shell commands | ✅ `bash` | ✅ `exec` | ✅ |
| Run tests | ✅ `bash` | ✅ `exec` | ✅ |
| Git operations | ✅ `bash` | ✅ `exec` | ✅ |

### Analysis Operations

| Capability | Claude Code | Ptahcortex | Status |
|------------|-------------|------------|--------|
| Code review | ✅ LLM | ✅ LLM | ✅ |
| Security audit | ✅ LLM | ✅ `audit` (Lexa) + LLM | 🏆 |
| Architecture analysis | ❌ | ✅ `audit` (Lexa) | 🏆 |
| Pipeline queries | ❌ | ✅ `pipeline` (Lexa) | 🏆 |

### Agent Operations

| Capability | Claude Code | Ptahcortex | Status |
|------------|-------------|------------|--------|
| Subagents | ✅ | ✅ | ✅ |
| Parallel execution | ✅ | ✅ | ✅ |
| Context isolation | ✅ | ✅ | ✅ |
| Streaming | ✅ | ❌ | ❌ |
| Hooks | ✅ | ❌ | ❌ |
| Memory (CLAUDE.md) | ✅ | ❌ | ❌ |

## Summary

### Ptahcortex Advantages (✅ Better)
- **Call graph analysis** — `callers` tool
- **Dependency tracing** — `trace_deps` tool
- **Architecture audit** — `audit` tool
- **Pipeline queries** — `pipeline` tool
- **Code intelligence** — 22 Lexa tools

### Claude Code Advantages (✅ Better)
- **Streaming** — Real-time output
- **Hooks** — Lifecycle automation
- **Memory** — CLAUDE.md persistence

### Equal (✅ Same)
- **Read/Write/Edit** — Both have full support
- **Execute** — Both run shell commands
- **Subagents** — Both support parallel execution
- **Analysis** — Both use LLM for code review

## Conclusion

**Ptahcortex has ALL read/write/analyze/execute capabilities:**
- ✅ Read: `read_file`, `read` (Lexa), `text_search`, `outline`, `symbol_search`
- ✅ Write: `write_file`, `patch` (Lexa), `create` (Lexa)
- ✅ Analyze: LLM + `audit` (Lexa) + `pipeline` (Lexa)
- ✅ Execute: `exec` (shell commands)

**Ptahcortex is MORE capable than Claude Code for code intelligence:**
- 🏆 Call graph analysis
- 🏆 Dependency tracing
- 🏆 Architecture audit
- 🏆 Pipeline queries

**Missing features (can add later):**
- ❌ Streaming
- ❌ Hooks
- ❌ Memory (CLAUDE.md)
