# Token Optimization & Prompt Quality Analysis

## Token Usage Comparison

### Ptahcortex (via Gateway)

| Request | Input | Output | Total | Purpose |
|---------|-------|--------|-------|---------|
| 1 | 3,194 | 290 | 3,484 | Final analysis |
| 2 | 380 | 28 | 408 | Subagent start |
| 3 | 3,196 | 96 | 3,292 | Main analysis |
| 4 | 380 | 11 | 391 | Subagent start |
| 5 | 253 | 50 | 303 | Initial test |
| **Total** | **7,403** | **475** | **7,878** | |

### Claude Code (Estimated)

| Phase | Input | Output | Total |
|-------|-------|--------|-------|
| Startup context | 2,000 | - | 2,000 |
| Tool calls | 3,000 | 500 | 3,500 |
| Analysis | 2,000 | 1,000 | 3,000 |
| **Total** | **7,000** | **1,500** | **8,500** |

## Token Optimization Issues

### 1. Large System Prompt

**Current Ptahcortex prompt includes:**
- Agent instructions (~500 tokens)
- Security checklist (~300 tokens)
- Tool definitions (~1,000 tokens)
- **Total: ~1,800 tokens per call**

**Claude Code:**
- Minimal system prompt (~200 tokens)
- Tool definitions deferred (~100 tokens)
- **Total: ~300 tokens per call**

### 2. Context Accumulation

**Current:** All results passed to analysis
```
Subagent 1 output: 500 tokens
Subagent 2 output: 500 tokens
Subagent 3 output: 500 tokens
Analysis prompt: 1,000 tokens
= 2,500 tokens for analysis
```

**Better:** Summarize before analysis
```
Subagent 1 summary: 100 tokens
Subagent 2 summary: 100 tokens
Subagent 3 summary: 100 tokens
Analysis prompt: 500 tokens
= 800 tokens for analysis
```

### 3. Tool Schema Overhead

**Current:** Full tool schemas sent every call
```
22 Lexa tools × 50 tokens = 1,100 tokens
```

**Better:** Only send relevant tools
```
3-5 relevant tools × 50 tokens = 150 tokens
```

## Prompt Quality Issues

### 1. Generic Security Checklist

**Current:**
```
CRITICAL SECURITY CHECKS:
1. JWT/Token Security:
   - Token in URL query parameters (Critical)
   - Weak/default JWT secrets (Critical)
   ...
```

**Better:** Task-specific prompts
```
Audit OAuth2 implementation in internal/auth/
Focus on: token storage, CSRF, redirect validation
```

### 2. No Code Context

**Current:** Generic analysis prompt
```
Analyze the following code for security vulnerabilities...
```

**Better:** Include file structure
```
Project structure:
- internal/auth/ (OAuth2 implementation)
- internal/middleware/ (security middleware)
- internal/config/ (configuration)

Analyze OAuth2 security in this context...
```

### 3. Output Format

**Current:** Free-form text
```
Provide security analysis with findings...
```

**Better:** Structured output
```
Return JSON:
{
  "findings": [
    {"file": "handler.go", "line": 84, "severity": "critical", "issue": "JWT in URL"}
  ],
  "patches": [...]
}
```

## Recommendations

### 1. Reduce System Prompt

```go
// Before: 1,800 tokens
prompt := "You are a senior security auditor. Analyze..."

// After: 300 tokens
prompt := "Analyze code for security issues. Return JSON."
```

### 2. Summarize Before Analysis

```go
// Before: Pass all results (2,500 tokens)
analysis := analyze(task, allResults)

// After: Summarize first (800 tokens)
summary := summarize(allResults)
analysis := analyze(task, summary)
```

### 3. Dynamic Tool Selection

```go
// Before: Send all tools (1,100 tokens)
tools := allLexaTools

// After: Send relevant tools (150 tokens)
tools := selectRelevantTools(task)
```

### 4. Structured Output

```go
// Before: Free text
"Provide security analysis..."

// After: JSON format
"Return JSON with findings, severity, patches..."
```

## Expected Improvements

| Metric | Current | Optimized | Savings |
|--------|---------|-----------|---------|
| System prompt | 1,800 | 300 | 83% |
| Context | 2,500 | 800 | 68% |
| Tool schemas | 1,100 | 150 | 86% |
| **Total** | **5,400** | **1,250** | **77%** |

## Conclusion

**Current token usage is inefficient:**
- ❌ Large system prompts
- ❌ Full context accumulation
- ❌ All tool schemas sent
- ❌ Free-form output

**Optimizations can reduce tokens by 77%:**
- ✅ Minimal prompts
- ✅ Summarize before analysis
- ✅ Dynamic tool selection
- ✅ Structured output
