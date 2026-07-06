#!/bin/bash
# Complex Benchmark: Multi-file Security Audit
# This task requires reading multiple files, tracing dependencies, and finding vulnerabilities

TASK="Perform a comprehensive security audit of the MCP client implementation. 

Specifically:
1. Trace the full authentication flow from client initialization to request signing
2. Identify all input validation gaps across MCP client, manager, and tool executor
3. Find race conditions in concurrent tool execution (parallel tools with semaphore)
4. Detect potential injection vulnerabilities in tool argument handling
5. Map all error handling paths and identify silent failures
6. Check for resource leaks (unclosed connections, goroutine leaks, file handles)
7. Verify TLS certificate validation and hostname verification

For each finding:
- Specify exact file and line numbers
- Rate severity (Critical/High/Medium/Low)
- Explain attack vector or failure scenario
- Suggest fix with code snippet

Files to analyze:
- internal/mcp/client.go
- internal/mcp/manager.go  
- internal/tools/executor.go
- internal/agent/agent.go
- internal/config/config.go"

echo "$TASK"
