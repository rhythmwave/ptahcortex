package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Tool represents a tool available on an MCP server.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Server      string         `json:"-"` // which server owns this tool
}

// ToolCall is a request to call a tool.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolResult is the result of a tool call.
type ToolResult struct {
	CallID  string `json:"call_id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// Client manages a single MCP server over stdio.
type Client struct {
	name    string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	mu      sync.Mutex
	nextID  int
	tools   []Tool
}

// New starts an MCP server process and initializes it.
func New(name, command string, args []string, cwd string) (*Client, error) {
	cmd := exec.Command(command, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mcp server %q: %w", name, err)
	}

	c := &Client{
		name:   name,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	// Initialize handshake
	if err := c.initialize(); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("initialize mcp %q: %w", name, err)
	}

	// Discover tools
	if err := c.discoverTools(); err != nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("discover tools on %q: %w", name, err)
	}

	return c, nil
}

// Tools returns the tools discovered from this server.
func (c *Client) Tools() []Tool {
	return c.tools
}

// CallTool calls a tool on this server.
func (c *Client) CallTool(name string, arguments map[string]any) (*ToolResult, error) {
	resp, err := c.request("tools/call", map[string]any{
		"name":      name,
		"arguments": arguments,
	})
	if err != nil {
		return nil, err
	}

	// Parse result
	result := &ToolResult{CallID: name}
	if resp.Result == nil {
		return result, fmt.Errorf("nil result from tool %q", name)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		return result, fmt.Errorf("unexpected result type: %T", resp.Result)
	}

	// Check isError
	if v, ok := resultMap["isError"].(bool); ok {
		result.IsError = v
	}

	// Extract content text
	if contents, ok := resultMap["content"].([]any); ok && len(contents) > 0 {
		if content, ok := contents[0].(map[string]any); ok {
			if text, ok := content["text"].(string); ok {
				result.Content = text
			}
		}
	}

	return result, nil
}

// Close shuts down the MCP server.
func (c *Client) Close() error {
	c.stdin.Close()
	return c.cmd.Process.Kill()
}

// Name returns the server name.
func (c *Client) Name() string {
	return c.name
}

// initialize does the MCP handshake.
func (c *Client) initialize() error {
	resp, err := c.request("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "agentkit", "version": "0.1.0"},
	})
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("initialize error: %v", resp.Error)
	}

	// Send initialized notification
	return c.notify("notifications/initialized", nil)
}

// discoverTools fetches the tool list from the server.
func (c *Client) discoverTools() error {
	resp, err := c.request("tools/list", nil)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %v", resp.Error)
	}

	resultMap, ok := resp.Result.(map[string]any)
	if !ok {
		return fmt.Errorf("unexpected tools/list result type: %T", resp.Result)
	}

	toolsJSON, err := json.Marshal(resultMap["tools"])
	if err != nil {
		return err
	}

	c.tools = nil
	if err := json.Unmarshal(toolsJSON, &c.tools); err != nil {
		return err
	}

	// Tag each tool with this server name
	for i := range c.tools {
		c.tools[i].Server = c.name
	}

	return nil
}

// --- JSON-RPC transport ---

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int   `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int   `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func (c *Client) request(method string, params any) (*jsonrpcResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.nextID++
	id := c.nextID

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Write request
	if _, err := fmt.Fprintf(c.stdin, "%s\n", data); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	// Read response
	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &resp, nil
}

func (c *Client) notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(c.stdin, "%s\n", data)
	return err
}
