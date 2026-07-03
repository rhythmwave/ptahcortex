# MCP Integration Guide

## What is MCP?

Model Context Protocol (MCP) is a standard for connecting AI models to external tools and data sources. It uses JSON-RPC 2.0 over stdio.

## How Ptahcortex Uses MCP

Ptahcortex connects to MCP servers as tool providers. The LLM doesn't know about MCP — it just sees tool definitions and calls them. Ptahcortex routes tool calls to the correct MCP server.

```
LLM says: "call search_code with query 'error handling'"
    │
    ▼
Ptahcortex routes to Lexa MCP server
    │
    ▼
Lexa returns: [{file: "main.go", line: 42, snippet: "..."}]
    │
    ▼
Ptahcortex formats result back to LLM
```

## MCP Protocol Flow

### 1. Initialize
```json
→ {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"ptahcortex","version":"0.1.0"}}}
← {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"serverInfo":{"name":"lexa","version":"0.6.1"}}}
```

### 2. List Tools
```json
→ {"jsonrpc":"2.0","id":2,"method":"tools/list"}
← {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"search_code","description":"Search code by query","inputSchema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}}]}}
```

### 3. Call Tool
```json
→ {"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search_code","arguments":{"query":"error handling"}}}
← {"jsonrpc":"2.0","id":3,"result":{"content":[{"type":"text","text":"Found 3 matches:\n1. main.go:42 - handleError(err)\n2. utils.go:15 - wrapError(msg, err)"}]}}
```

## Adding a New MCP Server

### Step 1: Write or find an MCP server

Any program that speaks MCP over stdio works. Examples:

- **Existing:** Lexa (code intel), filesystem server, web scraper
- **Custom:** Write your own in any language

### Step 2: Register in config

```yaml
# configs/mcp-servers.yaml
servers:
  my-server:
    description: My custom tools
    command: /path/to/server
    args: ["--flag", "value"]
    cwd: /working/dir
    env:
      API_KEY: "${MY_API_KEY}"  # env var substitution
```

### Step 3: Use in agent config

```yaml
# configs/agent.yaml
mcp_servers:
  - name: my-server
    # references mcp-servers.yaml
```

## Writing a Custom MCP Server

Minimal MCP server (Go):

```go
package main

import (
    "bufio"
    "encoding/json"
    "os"
)

type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      int             `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params"`
}

type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      int         `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   interface{} `json:"error,omitempty"`
}

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req Request
        json.Unmarshal(scanner.Bytes(), &req)
        
        var resp Response
        resp.JSONRPC = "2.0"
        resp.ID = req.ID
        
        switch req.Method {
        case "initialize":
            resp.Result = map[string]interface{}{
                "protocolVersion": "2024-11-05",
                "capabilities":   map[string]interface{}{"tools": map[string]interface{}{}},
                "serverInfo":     map[string]interface{}{"name": "my-server", "version": "0.1.0"},
            }
        case "tools/list":
            resp.Result = map[string]interface{}{
                "tools": []map[string]interface{}{
                    {
                        "name":        "my_tool",
                        "description": "Does something useful",
                        "inputSchema": map[string]interface{}{
                            "type":       "object",
                            "properties": map[string]interface{}{"input": map[string]interface{}{"type": "string"}},
                            "required":   []string{"input"},
                        },
                    },
                },
            }
        case "tools/call":
            // Parse params, execute tool, return result
            resp.Result = map[string]interface{}{
                "content": []map[string]interface{}{
                    {"type": "text", "text": "result here"},
                },
            }
        }
        
        data, _ := json.Marshal(resp)
        os.Stdout.Write(append(data, '\n'))
    }
}
```

## Available MCP Servers

| Server | Language | Tools | Use Case |
|---|---|---|---|
| Lexa | Rust | 22 | Code intelligence (search, symbols, references) |
| Filesystem | TypeScript | 6 | File read/write/search |
| Web Search | TypeScript | 2 | Web search and fetch |
| GitHub | TypeScript | 10 | GitHub API (issues, PRs, repos) |
| PostgreSQL | TypeScript | 3 | Database query |
| Custom | Any | Any | Your own tools |

## Tool Schema

MCP tools use JSON Schema for input validation:

```json
{
  "name": "search_code",
  "description": "Search codebase by natural language query",
  "inputSchema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Natural language search query"
      },
      "file_pattern": {
        "type": "string",
        "description": "Optional glob pattern to filter files"
      }
    },
    "required": ["query"]
  }
}
```

Ptahcortex validates tool arguments against this schema before calling the MCP server.

## Error Handling

MCP tool errors are returned as:

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{"type": "text", "text": "Error: file not found"}],
    "isError": true
  }
}
```

Ptahcortex treats `isError: true` results as tool errors and reports them to the LLM so it can adjust its approach.
