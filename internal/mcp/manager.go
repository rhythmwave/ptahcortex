package mcp

import (
	"fmt"
)

// Manager holds multiple MCP servers and routes tool calls.
type Manager struct {
	servers map[string]*Client
	tools   []Tool
}

// NewManager creates an empty manager.
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]*Client),
	}
}

// AddServer starts and registers an MCP server.
func (m *Manager) AddServer(name, command string, args []string, cwd string) error {
	client, err := New(name, command, args, cwd)
	if err != nil {
		return err
	}
	m.servers[name] = client

	// Aggregate tools
	m.tools = append(m.tools, client.Tools()...)
	return nil
}

// AllTools returns tools from all connected servers.
func (m *Manager) AllTools() []Tool {
	return m.tools
}

// CallTool routes a tool call to the correct server.
func (m *Manager) CallTool(name string, arguments map[string]any) (*ToolResult, error) {
	// Find which server owns this tool
	for _, t := range m.tools {
		if t.Name == name {
			return m.servers[t.Server].CallTool(name, arguments)
		}
	}
	return nil, fmt.Errorf("tool %q not found on any server", name)
}

// Close shuts down all servers.
func (m *Manager) Close() {
	for _, c := range m.servers {
		c.Close()
	}
}
