package tools

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// BasicTool provides basic OS operations (Read, Write, Execute)
type BasicTool struct {
	WorkDir string
}

// NewBasicTool creates a new basic tool with working directory
func NewBasicTool(workDir string) *BasicTool {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}
	return &BasicTool{WorkDir: workDir}
}

// ReadFile reads a file
func (t *BasicTool) ReadFile(path string) (string, error) {
	// Resolve path
	if !strings.HasPrefix(path, "/") {
		path = t.WorkDir + "/" + path
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	return string(data), nil
}

// WriteFile writes to a file
func (t *BasicTool) WriteFile(path, content string) error {
	// Resolve path
	if !strings.HasPrefix(path, "/") {
		path = t.WorkDir + "/" + path
	}
	
	// Create directory if needed
	dir := path[:strings.LastIndex(path, "/")]
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}
	
	return os.WriteFile(path, []byte(content), 0644)
}

// Exec runs a shell command
func (t *BasicTool) Exec(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = t.WorkDir
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("exec: %w", err)
	}
	return string(output), nil
}

// ListFiles lists files in a directory
func (t *BasicTool) ListFiles(path string) (string, error) {
	if path == "" {
		path = t.WorkDir
	}
	if !strings.HasPrefix(path, "/") {
		path = t.WorkDir + "/" + path
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("list files: %w", err)
	}
	
	var result []string
	for _, entry := range entries {
		if entry.IsDir() {
			result = append(result, entry.Name()+"/")
		} else {
			result = append(result, entry.Name())
		}
	}
	return strings.Join(result, "\n"), nil
}
