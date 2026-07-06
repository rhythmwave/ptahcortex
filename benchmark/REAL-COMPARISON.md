# Real Comparison: Aider vs Ptahcortex

## Task: Security Audit (5 files)

**Task**: "Security audit of MCP client and tool executor. Find race conditions, resource leaks, and injection vulnerabilities across all files."

**Files**: 5 files (client.go, manager.go, executor.go, agent.go, config.go)

## Results

### Aider (gpt-4.1-mini)
```
Tokens:  11,000 sent + 656 received = ~11,656 tokens
Duration: ~30 seconds
Findings: 2 issues (race condition, resource leak)
Output: Analysis + offered code patches
Cost: $0.0053
```

### Ptahcortex (glm-5.1)
```
Iteration 1: 3,517 tokens (59s)
Iteration 2: 8,705 tokens (1m4s)
Iteration 3: 16,367 tokens (1m56s)
Iteration 4: 22,318 tokens (partial, killed)
Total: ~22,318 tokens (incomplete)
Duration: 4+ minutes (killed)
Findings: Partial (reading files)
Cost: ~$0.03
```

## Comparison

| Metric | Aider | Ptahcortex |
|--------|-------|------------|
| **Tokens** | 11,656 | 22,318+ (incomplete) |
| **Duration** | 30s | 4m+ (killed) |
| **Findings** | 2 complete | Partial |
| **Cost** | $0.005 | $0.03 |
| **Status** | ✅ Complete | ❌ Killed |

## Why Aider Won

1. **Single pass**: Read all 5 files at once, analyze in one LLM call
2. **No sandbox overhead**: Direct tool calling
3. **Efficient**: 11k tokens vs 22k+ tokens

## Why Ptahcortex Lost

1. **Sandbox overhead**: Each file read = 2 LLM calls (select + evaluate)
2. **Multi-iteration**: Keeps reading files across iterations
3. **Token accumulation**: Summaries grow each iteration

## The Real Problem

Ptahcortex's sandboxed architecture:
```
For EACH file read:
1. LLM selects tool (2,350 tokens)
2. MCP reads file (0 tokens)
3. LLM evaluates result (3,700 tokens)
4. LLM summarizes (1,000 tokens)
Total per file: ~7,050 tokens

For 5 files: 35,250 tokens
```

Aider's approach:
```
For ALL files:
1. Read files locally (0 tokens)
2. Single LLM call (11,656 tokens)
Total: 11,656 tokens
```

## Conclusion

Aider wins because:
- Reads files locally (no LLM calls)
- Single LLM call for analysis
- No sandbox overhead

Ptahcortex loses because:
- LLM calls for every tool operation
- Sandbox adds overhead
- Multi-iteration wastes tokens

## The Fix

Ptahcortex needs:
1. **Local file reading** (bypass MCP for reads)
2. **Single LLM call** (not iterative)
3. **Remove sandbox** for simple tasks
4. **Keep sandbox** for complex reasoning only

**Result**: Aider-like efficiency + Ptahcortex depth + OTel observability
