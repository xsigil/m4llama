package m4

import (
	"strings"
	"testing"
)

func TestEngine_BasicMacros(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple define and expand",
			input:    "define(`FOO', `bar')FOO",
			expected: "bar",
		},
		{
			name:     "macro with arguments",
			input:    "define(`GREET', `Hello, $1!')GREET(`World')",
			expected: "Hello, World!",
		},
		{
			name:     "dnl removes trailing newline",
			input:    "define(`FOO', `bar')dnl\nFOO",
			expected: "bar",
		},
		{
			name:     "ifdef defined",
			input:    "define(`FLAG', `1')ifdef(`FLAG', `YES', `NO')",
			expected: "YES",
		},
		{
			name:     "ifdef undefined",
			input:    "ifdef(`FLAG', `YES', `NO')",
			expected: "NO",
		},
		{
			name:     "ifelse match",
			input:    "ifelse(`a', `a', `match', `mismatch')",
			expected: "match",
		},
		{
			name:     "ifelse mismatch",
			input:    "ifelse(`a', `b', `match', `mismatch')",
			expected: "mismatch",
		},
		{
			name:     "nested quotes preservation",
			input:    "define(`WRAP', `[$1]')WRAP(```inner quote''')",
			expected: "[`inner quote']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine()
			got, err := engine.Expand(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEngine_Diversion(t *testing.T) {
	input := strings.TrimSpace(`
divert(1)dnl
This is diverted text.
divert(0)dnl
Initial text.
undivert(1)dnl
`)

	engine := NewEngine()
	got, err := engine.Expand(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Initial text.\nThis is diverted text.\n"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
