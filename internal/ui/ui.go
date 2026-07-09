package ui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"golang.org/x/term"
)

const (
	// Colors
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	BgBlue  = "\033[44m"
)

// Banner shows the Ptahcortex banner
func Banner(version, model, effort string) {
	width := getTermWidth()
	
	fmt.Println()
	fmt.Printf("%s%s▐▛███▜▌%s Ptahcortex %s%s%s\n", Bold, Cyan, Reset, Green, version, Reset)
	fmt.Printf("%s▝▜█████▛▘%s %s%s%s with %s%s%s effort\n", Cyan, Reset, Yellow, model, Reset, Magenta, effort, Reset)
	fmt.Printf("%s ▘▘ ▝▝%s %s\n", Dim, Reset, getProjectPath())
	fmt.Println()
	fmt.Printf("%s%s%s\n", Dim, strings.Repeat("─", width), Reset)
	fmt.Println()
}

// Prompt shows the input prompt
func Prompt() string {
	return fmt.Sprintf("%s%s❯%s ", Bold, Green, Reset)
}

// Thinking shows a thinking indicator
func Thinking(seconds int, actions ...string) {
	fmt.Printf("%s%s● Thinking for %ds%s", Dim, Yellow, seconds, Reset)
	
	if len(actions) > 0 {
		fmt.Printf(", %s", strings.Join(actions, ", "))
	}
	
	fmt.Println()
}

// ToolCall shows a tool being executed
func ToolCall(tool string, args string, duration string) {
	fmt.Printf("  %s├─%s %s%s%s", Dim, Reset, Cyan, tool, Reset)
	if args != "" {
		fmt.Printf(" %s%s%s", Dim, args, Reset)
	}
	if duration != "" {
		fmt.Printf(" %s(%s)%s", Dim, duration, Reset)
	}
	fmt.Println()
}

// ToolResult shows a tool result
func ToolResult(result string, lines int) {
	fmt.Printf("  %s└─%s %s%d lines%s\n", Dim, Reset, Green, lines, Reset)
}

// Error shows an error message
func Error(msg string) {
	fmt.Printf("%s%s✖ %s%s\n", Bold, Red, msg, Reset)
}

// Success shows a success message
func Success(msg string) {
	fmt.Printf("%s%s✔ %s%s\n", Bold, Green, msg, Reset)
}

// Warning shows a warning message
func Warning(msg string) {
	fmt.Printf("%s%s⚠ %s%s\n", Bold, Yellow, msg, Reset)
}

// Info shows an info message
func Info(msg string) {
	fmt.Printf("%s%sℹ %s%s\n", Bold, Blue, msg, Reset)
}

// Progress shows a progress bar
func Progress(current, total int) {
	width := 30
	filled := (current * width) / total
	
	fmt.Printf("\r%s[%s%s%s] %d/%d%s", 
		Dim,
		Green, strings.Repeat("█", filled), Reset,
		current, total,
		Reset,
	)
	
	if current == total {
		fmt.Println()
	}
}

// Table shows a formatted table
func Table(headers []string, rows [][]string) {
	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	
	// Print header
	fmt.Printf("%s%s", Bold, White)
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], h)
	}
	fmt.Println(Reset)
	
	// Print separator
	for i := range headers {
		fmt.Printf("%s%s%s", Dim, strings.Repeat("─", widths[i]), Reset)
		fmt.Print("  ")
	}
	fmt.Println()
	
	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			fmt.Printf("%-*s  ", widths[i], cell)
		}
		fmt.Println()
	}
}

// Status shows agent status
func Status(agent, model, tokens int, duration string) {
	fmt.Printf("%s%s┌─ Status ─────────────────────┐%s\n", Dim, White, Reset)
	fmt.Printf("%s│%s Agent:    %s%-20s%s%s│%s\n", Dim, Reset, Green, agent, Dim, Reset, Reset)
	fmt.Printf("%s│%s Model:    %s%-20s%s%s│%s\n", Dim, Reset, Yellow, model, Dim, Reset, Reset)
	fmt.Printf("%s│%s Tokens:   %s%-20d%s%s│%s\n", Dim, Reset, Cyan, tokens, Dim, Reset, Reset)
	fmt.Printf("%s│%s Duration: %s%-20s%s%s│%s\n", Dim, Reset, Magenta, duration, Dim, Reset, Reset)
	fmt.Printf("%s%s└─────────────────────────────┘%s\n", Dim, White, Reset)
}

// getTermWidth returns terminal width
func getTermWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

// getProjectPath returns current directory
func getProjectPath() string {
	dir, err := os.Getwd()
	if err != nil {
		return "~"
	}
	
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(dir, home) {
		return "~" + dir[len(home):]
	}
	
	return dir
}

// GetVersion returns the version string
func GetVersion() string {
	return "v0.1.0"
}

// GetOS returns the OS info
func GetOS() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
