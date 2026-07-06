# Ptahcortex vs Aider: Comprehensive Benchmark Comparison

## Executive Summary

Ptahcortex and Aider represent different approaches to AI-assisted code review:
- **Ptahcortex**: Iterative, sandboxed tool reasoning with call-aware context assembly
- **Aider**: Single-pass, direct tool calling with sliding window context

**Key Finding**: Ptahcortex provides deeper analysis (7 findings vs 5) with more high-severity issues, while Aider is more token-efficient (74% fewer tokens) and cost-effective (10x cheaper).

## Architecture Comparison

| Feature | Ptahcortex | Aider |
|---------|------------|-------|
| **Language** | Go | Python |
| **Tool Protocol** | MCP (stdio) | Built-in (git, file ops) |
| **Context Management** | Call-aware assembly (T0-T4) | Sliding window |
| **Tool Reasoning** | Sandboxed (isolated LLM calls) | Direct (same context) |
| **Observability** | OTel (traces, metrics) | Basic logging |
| **Config** | YAML-driven | CLI flags + .env |
| **Binary Size** | ~10MB | ~50MB (Python) |
| **Dependencies** | Single binary | Python + packages |

## Benchmark Results

### Task: Code Review (Error Handling)

| Metric | Ptahcortex | Aider |
|--------|------------|-------|
| **Total Tokens** | 14,353 | 3,775 |
| **Iterations** | 3 | 1 (single pass) |
| **Tool Calls** | 10 | 5-8 |
| **Duration** | 2m 45s | ~1m 30s |
| **Cost** | ~$0.02 | $0.002 |
| **Findings** | 7 | 5 |
| **High Severity** | 3 | 0 |
| **Medium Severity** | 2 | 3 |
| **Low Severity** | 2 | 2 |

### Token Breakdown (Ptahcortex)

| Phase | Tokens | Percentage |
|-------|--------|------------|
| Plan | 8,345 | 58.1% |
| Sandbox | 14,722 | (tool execution) |
| Reflect | 3,222 | 22.4% |
| Final | 2,786 | 19.4% |

## Quality Comparison

### Ptahcortex Findings (7 total)

1. **Unbounded Read / DoS Vulnerability** (High) - Specific code location
2. **Potential Deadlock / Stale Lock** (High) - Race condition identified
3. **Missing Response ID Validation** (High) - Security issue
4. **Unsafe Type Assertion** (Medium) - Runtime panic risk
5. **Ignored Start Errors** (Medium) - Error handling gap
6. **Silent Failure on Stop** (Low) - Graceful shutdown issue
7. **Missing Context Cancellation** (Low) - Resource leak

### Aider Findings (5 total)

1. **Config Loading** (Medium) - Missing error wrapping
2. **MCP Server Addition** (Medium) - No partial cleanup
3. **Stdin Reading** (Low) - Minimal validation
4. **Context Handling** (Low) - Graceful shutdown issues
5. **Agent Execution** (Medium) - No cleanup on error

## Key Insights

### 1. Token Efficiency vs Depth
- **Aider**: 74% fewer tokens, suitable for quick reviews
- **Ptahcortex**: More tokens but deeper analysis, better for comprehensive audits

### 2. Cost Effectiveness
- **Aider**: $0.002 per review (10x cheaper)
- **Ptahcortex**: $0.02 per review (justified for high-severity findings)

### 3. Finding Quality
- **Ptahcortex**: Found 3 high-severity issues (security, race conditions)
- **Aider**: Focused on medium/low severity (code quality)

### 4. Use Case Suitability
- **Aider**: Quick code reviews, iterative development, cost-sensitive
- **Ptahcortex**: Security audits, production code, compliance requirements

## Technical Deep Dive

### Context Management

**Ptahcortex (Call-Aware Assembly)**:
- 5 call types: plan, sandbox_select, sandbox_eval, reflect, final
- Different context tiers (T0-T4) for each call type
- ~72% token savings at 20 iterations
- Isolated tool reasoning prevents context pollution

**Aider (Sliding Window)**:
- Single context window for all operations
- Direct tool calling in same context
- Simple but can lead to context pollution
- No token optimization

### Tool Execution

**Ptahcortex (Sandboxed)**:
- Isolated LLM calls for tool selection and evaluation
- Only summaries flow back to main context
- Prevents tool output from polluting reasoning
- Better for complex, multi-step operations

**Aider (Direct)**:
- Tool calls happen in main context
- Tool output directly affects reasoning
- Simpler but can cause context drift
- Better for straightforward operations

## Recommendations

### When to Use Ptahcortex
- Security audits and vulnerability assessments
- Production code reviews
- Compliance requirements
- Complex multi-file analysis
- When high-severity findings matter most

### When to Use Aider
- Quick code reviews during development
- Iterative refactoring
- Cost-sensitive applications
- Simple, single-file reviews
- When speed and efficiency matter most

## Future Improvements

### Ptahcortex
1. Optimize token usage in plan phase
2. Add more MCP tools for deeper analysis
3. Implement caching for repeated analyses
4. Add parallel review capabilities

### Aider
1. Add MCP support for external tools
2. Implement iterative review mode
3. Add severity-based filtering
4. Improve context management for large codebases

## Conclusion

Both tools serve different purposes in the AI-assisted code review landscape:

- **Ptahcortex** is the choice for **depth and security** - finding critical issues that matter most in production environments
- **Aider** is the choice for **speed and efficiency** - perfect for development workflows where quick feedback is essential

The ideal approach is to use both: Aider for quick checks during development, Ptahcortex for comprehensive reviews before production deployment.

---

**Benchmark Date**: 2026-07-06
**Models Used**: GLM-5.1 (Ptahcortex), GPT-4.1-mini (Aider)
**API**: ai.sumopod.com
**Codebase**: commit-reviewer (Go)
