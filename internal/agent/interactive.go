package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rhythmwave/ptahcortex/internal/config"
	"github.com/rhythmwave/ptahcortex/internal/llm"
	"github.com/rhythmwave/ptahcortex/internal/mcp"
	"github.com/rhythmwave/ptahcortex/internal/tools"
)

// InteractiveAgent maintains context across multiple user inputs
type InteractiveAgent struct {
	cfg      *config.Config
	llm      llm.Provider
	mcp      *mcp.Manager
	basic    *tools.BasicTool
	useLexa  bool
	
	// Conversation history
	messages []llm.Message
	
	// Context from previous turns
	previousResults map[string]string
	turnCount       int
}

// NewInteractiveAgent creates an agent that maintains context
func NewInteractiveAgent(cfg *config.Config, provider llm.Provider, mcpManager *mcp.Manager, useLexa bool) *InteractiveAgent {
	return &InteractiveAgent{
		cfg:             cfg,
		llm:             provider,
		mcp:             mcpManager,
		basic:           tools.NewBasicTool(""),
		useLexa:         useLexa,
		messages:        []llm.Message{},
		previousResults: make(map[string]string),
		turnCount:       0,
	}
}

// RunInteractive starts an interactive session
func (a *InteractiveAgent) RunInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)
	
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("  Ptahcortex Interactive Mode")
	fmt.Println("  Type 'exit' or 'quit' to end session")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()
	
	// Initialize with system prompt
	a.messages = append(a.messages, llm.Message{
		Role: "system",
		Content: `You are Ptahcortex, a code analysis agent. You have access to tools for reading files, executing commands, and searching code.

IMPORTANT: Always use the available tools to complete tasks. Do NOT give generic advice - actually use the tools to find and analyze code.

When the user asks you to:
- "List files" → use list_files or exec with find command
- "Find code" → use search tool
- "Read file" → use read_file tool
- "Analyze" → use tools first, then analyze results`,
	})
	
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		
		if input == "exit" || input == "quit" {
			fmt.Println("Session ended.")
			break
		}
		
		a.turnCount++
		log.Printf("\n[interactive] Turn %d: %s", a.turnCount, input)
		
		// Process the user input
		response, err := a.processTurn(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		
		fmt.Printf("\nAgent: %s\n\n", response)
	}
	
	return nil
}

// processTurn handles a single user input with context from previous turns
func (a *InteractiveAgent) processTurn(input string) (string, error) {
	// Build context from previous turns
	context := a.buildContext()
	
	// Create prompt with context
	prompt := a.buildPrompt(input, context)
	
	// Add user message to history
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: prompt,
	})
	
	// Call LLM with tools
	resp, err := a.llm.Chat(llm.ChatRequest{
		Messages:   a.messages,
		Tools:      a.buildTools(),
		MaxTokens:  a.cfg.LLM.MaxTokens,
		Model:      a.cfg.LLM.Model,
	})
	if err != nil {
		return "", err
	}
	
	log.Printf("[interactive] LLM response - Content length: %d, ToolCalls: %d", len(resp.Content), len(resp.ToolCalls))
	
	// Execute tool calls if any
	if len(resp.ToolCalls) > 0 {
		// Add assistant message with tool_calls first
		a.messages = append(a.messages, llm.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})
		
		for _, tc := range resp.ToolCalls {
			log.Printf("[interactive] executing tool: %s", tc.Function.Name)
			result := a.executeToolCall(tc)
				
			// Add tool result to messages
			a.messages = append(a.messages, llm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
		
		// Get final response after tool execution
		log.Printf("[interactive] Calling LLM with %d messages after tool execution", len(a.messages))
		finalResp, err := a.llm.Chat(llm.ChatRequest{
			Messages:   a.messages,
			MaxTokens:  a.cfg.LLM.MaxTokens,
			Model:      a.cfg.LLM.Model,
		})
		if err != nil {
			return "", err
		}
		log.Printf("[interactive] Final response - Content length: %d, ToolCalls: %d", len(finalResp.Content), len(finalResp.ToolCalls))
		resp = finalResp
	}
	

	
	// Add assistant response to history
	a.messages = append(a.messages, llm.Message{
		Role:    "assistant",
		Content: resp.Content,
	})
	
	// Store results for context
	a.previousResults[fmt.Sprintf("turn_%d", a.turnCount)] = resp.Content
	
	return resp.Content, nil
}

// executeToolCall executes a single tool call
func (a *InteractiveAgent) executeToolCall(tc llm.ToolCall) string {
	var args map[string]any
	json.Unmarshal([]byte(tc.Function.Arguments), &args)
	
	var result string
	var err error
	
	switch tc.Function.Name {
	case "read_file":
		path, _ := args["path"].(string)
		result, err = a.basic.ReadFile(path)
	case "exec":
		command, _ := args["command"].(string)
		result, err = a.basic.Exec(command)
	case "list_files":
		path, _ := args["path"].(string)
		result, err = a.basic.ListFiles(path)
	case "search":
		if a.useLexa {
			query, _ := args["query"].(string)
			r, e := a.mcp.CallTool("text_search", map[string]any{"query": query})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		}
	case "outline":
		if a.useLexa {
			path, _ := args["path"].(string)
			r, e := a.mcp.CallTool("outline", map[string]any{"path": path})
			if e != nil {
				err = e
			} else {
				result = r.Content
			}
		}
	default:
		result = fmt.Sprintf("Unknown tool: %s", tc.Function.Name)
	}
	
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}
	
	return result
}

// buildTools returns available tools
func (a *InteractiveAgent) buildTools() []llm.ToolDefinition {
	tools := []llm.ToolDefinition{
		{Type: "function", Function: llm.ToolFunction{
			Name: "read_file", Description: "Read file contents",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{"path": map[string]any{"type": "string"}},
				"required": []string{"path"},
			},
		}},
		{Type: "function", Function: llm.ToolFunction{
			Name: "exec", Description: "Execute shell command",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{"command": map[string]any{"type": "string"}},
				"required": []string{"command"},
			},
		}},
		{Type: "function", Function: llm.ToolFunction{
			Name: "list_files", Description: "List files in directory",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{"path": map[string]any{"type": "string"}},
			},
		}},
	}
	
	if a.useLexa {
		tools = append(tools,
			llm.ToolDefinition{Type: "function", Function: llm.ToolFunction{
				Name: "search", Description: "Search code patterns",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{"query": map[string]any{"type": "string"}},
					"required": []string{"query"},
				},
			}},
			llm.ToolDefinition{Type: "function", Function: llm.ToolFunction{
				Name: "outline", Description: "Get file structure",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{"path": map[string]any{"type": "string"}},
					"required": []string{"path"},
				},
			}},
		)
	}
	
	return tools
}

// buildContext builds context from previous turns
func (a *InteractiveAgent) buildContext() string {
	if len(a.previousResults) == 0 {
		return ""
	}
	
	var context strings.Builder
	context.WriteString("Previous conversation:\n")
	
	for turn, result := range a.previousResults {
		// Truncate long results
		truncated := result
		if len(truncated) > 300 {
			truncated = truncated[:300] + "..."
		}
		context.WriteString(fmt.Sprintf("\n[%s]\n%s\n", turn, truncated))
	}
	
	return context.String()
}

// buildPrompt builds the prompt with context
func (a *InteractiveAgent) buildPrompt(input string, context string) string {
	if context == "" {
		return input
	}
	
	return fmt.Sprintf(`%s

Context from previous conversation:
%s

Current request: %s`, input, context, input)
}
