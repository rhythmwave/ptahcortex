# Benchmark Summary: Ptahcortex vs Aider

## Date: 2026-07-06

## What We Did

1. **Installed Aider** - Python-based coding agent with OpenAI-compatible API support
2. **Configured API** - Used ai.sumopod.com with GLM-5.1 and GPT-4.1-mini models
3. **Ran Benchmark** - Code review task on commit-reviewer codebase
4. **Analyzed Results** - Comprehensive comparison of architecture, performance, and quality

## Key Results

| Metric | Ptahcortex | Aider |
|--------|------------|-------|
| **Tokens** | 14,353 | 3,775 |
| **Findings** | 7 | 5 |
| **High Severity** | 3 | 0 |
| **Cost** | ~$0.02 | $0.002 |
| **Architecture** | Sandboxed tool reasoning | Direct tool calling |

## Insights

### Ptahcortex Strengths
- **Deeper Analysis**: Found 3 high-severity issues (security, race conditions)
- **Better Quality**: More specific, actionable findings
- **Novel Architecture**: Call-aware context assembly is unique in OSS
- **Observability**: Full OTel integration for debugging

### Aider Strengths
- **Token Efficiency**: 74% fewer tokens used
- **Cost Effective**: 10x cheaper per review
- **Simpler Architecture**: Easier to understand and maintain
- **Fast**: Single-pass execution

## Use Cases

### When to Use Ptahcortex
- Security audits and vulnerability assessments
- Production code reviews
- Compliance requirements
- Complex multi-file analysis

### When to Use Aider
- Quick code reviews during development
- Iterative refactoring
- Cost-sensitive applications
- Simple, single-file reviews

## Files Created

- `benchmark/AIDER-RESULT.md` - Aider benchmark result
- `benchmark/BENCHMARK-COMPARISON.md` - Quick comparison
- `benchmark/COMPREHENSIVE-COMPARISON.md` - Detailed analysis

## Conclusion

Both tools serve different purposes:
- **Ptahcortex** = Depth and security (finding critical issues)
- **Aider** = Speed and efficiency (quick feedback)

The ideal approach: Use Aider for development, Ptahcortex for production.

## Next Steps

1. Run full benchmark suite (5 tasks) with both agents
2. Test with different models (GLM-5.2, MiniMax)
3. Add more comparison agents (Crush, Codex)
4. Generate comprehensive report for portfolio
