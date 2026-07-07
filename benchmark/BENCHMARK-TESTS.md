# Ptahcortex Benchmark Suite

## Test Scenarios

### Scenario 1: Security Audit (Existing)
**Task:** Audit OAuth2 implementation for security vulnerabilities
**Expected:** Find vulnerabilities, generate patches
**Tools:** text_search, outline, read, audit

### Scenario 2: Code Generation
**Task:** Generate a new function to validate JWT tokens with proper error handling
**Expected:** Generate code with proper validation, error handling, tests
**Tools:** text_search, outline, read

### Scenario 3: Bug Investigation
**Task:** Find and fix the race condition in concurrent API calls
**Expected:** Identify race condition, suggest fix with mutex/channel
**Tools:** text_search, outline, read, callers

### Scenario 4: Refactoring
**Task:** Refactor the authentication module to be more modular
**Expected:** Suggest code structure, identify dependencies
**Tools:** outline, read, callers, trace_deps

### Scenario 5: Documentation
**Task:** Generate API documentation for the auth endpoints
**Expected:** Generate OpenAPI spec or markdown docs
**Tools:** outline, read, text_search

## Benchmark Metrics

| Metric | Description |
|--------|-------------|
| **Tokens** | Total LLM tokens used |
| **Duration** | Wall-clock time |
| **Tool Calls** | Number of tool invocations |
| **Quality** | Correctness of output (1-5) |
| **Cost** | Estimated cost in USD |

## Comparison Agents

| Agent | Type | Tokens | Duration |
|-------|------|--------|----------|
| Ptahcortex (Smart) | LLM planned | ~1.5k | ~1 min |
| Ptahcortex (Auto) | Rule-based | ~3k | ~46s |
| Aider | Direct editing | ~11k | ~30s |
| Claude Code | Full agent | ~100k+ | ~5 min |
