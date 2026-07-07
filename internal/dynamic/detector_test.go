package dynamic

import (
	"fmt"
	"testing"
)

func TestDetector(t *testing.T) {
	// Create detector with test config
	config := &Config{
		Mappings: map[string]*Mapping{
			"security": {
				Description: "Security audit",
				Tools:       []string{"text_search", "callers", "trace_deps"},
				Keywords:    []string{"race condition", "mutex", "injection"},
			},
			"auth": {
				Description: "Auth analysis",
				Tools:       []string{"text_search", "outline", "read"},
				Keywords:    []string{"oauth", "pkce", "token", "session"},
			},
		},
		Commands: map[string]string{
			"/security": "security",
			"/auth":     "auth",
		},
	}

	detector := &Detector{config: config}

	tests := []struct {
		name     string
		task     string
		expected string
	}{
		{
			name:     "command security",
			task:     "/security audit src/auth/",
			expected: "security",
		},
		{
			name:     "command auth",
			task:     "/auth check login flow",
			expected: "auth",
		},
		{
			name:     "keyword race condition",
			task:     "Find race conditions in concurrent code",
			expected: "security",
		},
		{
			name:     "keyword oauth",
			task:     "Analyze OAuth2 implementation",
			expected: "auth",
		},
		{
			name:     "default",
			task:     "Review this code",
			expected: "review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.Detect(tt.task)
			if result.Category != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Category)
			}
			fmt.Printf("✓ %s: %s (source: %s)\n", tt.name, result.Category, result.Source)
		})
	}
}
