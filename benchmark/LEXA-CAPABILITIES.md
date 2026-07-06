# Lexa Capabilities (No LLM Required)

## What Lexa Already Has

### 1. **Graph Indexing** ✅
- 119 files indexed
- Dependency graph built
- Symbol extraction complete

### 2. **Search Tools** ✅
- `text_search(query)` - Find patterns
- `search(pattern)` - Advanced search
- `glob(pattern)` - File patterns

### 3. **Code Analysis** ✅
- `outline(path)` - File structure
- `callers(symbol)` - Who calls what
- `trace_deps(path)` - Dependency chain

### 4. **Pipeline** ✅
- Composable queries
- Chain multiple operations
- Filter and limit results

### 5. **Audit** ✅
- Architecture analysis
- Code quality findings
- Dependency hotspots

## Test Results

### Search for Security Patterns
```
exec.Command: 2 results
mutex: 3 results  
go func: 7 results
```

### Audit Findings
- [High] Large method `Execute` (263 lines)
- [Warning] Large function `SettingsScreen` (122 lines)
- [Warning] Dependency hotspot `api.ts`

### Pipeline Queries
```
search exec.Command | search mutex | search go func | limit 20
→ 7 results (all goroutine usages)
```

## What Ptahcortex Needs

### Current (Wasteful)
```
LLM Plan → Lexa → LLM Evaluate → LLM Summarize
= 3 LLM calls + Lexa
```

### New (Efficient)
```
Lexa Pipeline (no LLM) → Single LLM Call
= 0 LLM calls for queries + 1 LLM call for analysis
```

## Implementation

### Step 1: Build Context with Lexa Pipeline
```go
func buildContext(task string) *Context {
    // 1. Extract keywords from task (no LLM)
    keywords := extractKeywords(task)
    
    // 2. Run Lexa pipeline (no LLM)
    pipeline := []string{}
    for _, kw := range keywords {
        pipeline = append(pipeline, fmt.Sprintf("search %s", kw))
    }
    pipeline = append(pipeline, "limit 50")
    
    results := mcp.Call("pipeline", pipeline)
    
    // 3. Get outlines for found files (no LLM)
    files := parseFiles(results)
    outlines := map[string]string{}
    for _, file := range files {
        outlines[file] = mcp.Call("outline", file)
    }
    
    // 4. Get dependencies (no LLM)
    deps := map[string]string{}
    for _, file := range files {
        deps[file] = mcp.Call("trace_deps", file)
    }
    
    return &Context{
        SearchResults: results,
        Outlines: outlines,
        Dependencies: deps,
    }
}
```

### Step 2: Single LLM Call
```go
func analyze(task string, ctx *Context) *Result {
    // Build prompt with Lexa results
    prompt := fmt.Sprintf(`
Task: %s

Search Results:
%s

Code Structure:
%s

Dependencies:
%s

Provide security analysis with:
1. Findings (severity + line numbers)
2. Code patches in diff format
`, task, ctx.SearchResults, ctx.Outlines, ctx.Dependencies)
    
    // Single LLM call
    return llm.Analyze(prompt)
}
```

## Token Comparison

### Current
```
LLM Plan: 2,541 tokens
Lexa: 0 tokens
LLM Evaluate: 3,700 tokens × 5 = 18,500 tokens
LLM Summarize: 1,000 tokens × 5 = 5,000 tokens
────────────────────────────
Total: 26,041 tokens
```

### New
```
Lexa Pipeline: 0 tokens
Lexa Outline: 0 tokens
Lexa TraceDeps: 0 tokens
LLM Analyze: 11,656 tokens (single call)
────────────────────────────
Total: 11,656 tokens
Savings: 55%
```

## Benefits

1. **Zero token cost** for all queries
2. **No LLM dependency** for planning
3. **Fast** (Lexa is local)
4. **Same depth** (full context)
5. **Keep OTel** (observability)

## Conclusion

Lexa already has everything we need:
- Graph ranking ✅
- Dependency analysis ✅
- Pattern search ✅
- Pipeline queries ✅
- Audit findings ✅

**We just need to use it directly without LLM in between!**
