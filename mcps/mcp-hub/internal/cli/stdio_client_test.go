package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetStdioCommandLength(t *testing.T) {
	tests := []struct {
		name     string
		osArgs   []string
		expected int
	}{
		{
			name:     "no dash dash",
			osArgs:   []string{"hub", "--stdio", "list"},
			expected: 0,
		},
		{
			name:     "with dash dash and command",
			osArgs:   []string{"hub", "--stdio", "list", "--", "npx", "@mcp/server"},
			expected: 2,
		},
		{
			name:     "with dash dash and single command",
			osArgs:   []string{"hub", "--stdio", "list", "--", "npx"},
			expected: 1,
		},
		{
			name:     "with dash dash but no command",
			osArgs:   []string{"hub", "--stdio", "list", "--"},
			expected: 0,
		},
		{
			name:     "dash dash with complex args",
			osArgs:   []string{"hub", "--stdio", "invoke", "echo", "{}", "--", "npx", "@mcp/server", "arg1", "arg2"},
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			os.Args = tt.osArgs
			result := getStdioCommandLength()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterArgsBeforeDash(t *testing.T) {
	tests := []struct {
		name     string
		osArgs   []string
		args     []string
		expected []string
	}{
		{
			name:     "no stdio command",
			osArgs:   []string{"hub", "list"},
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "with stdio command - filter out",
			osArgs:   []string{"hub", "--stdio", "inspect", "echo", "--", "npx", "@mcp/server"},
			args:     []string{"echo", "npx", "@mcp/server"},
			expected: []string{"echo"},
		},
		{
			name:     "with stdio command - invoke with params",
			osArgs:   []string{"hub", "--stdio", "invoke", "echo", "{}", "--", "npx", "@mcp/server"},
			args:     []string{"echo", "{}", "npx", "@mcp/server"},
			expected: []string{"echo", "{}"},
		},
		{
			name:     "no dash dash in osArgs",
			osArgs:   []string{"hub", "-u", "http://localhost", "list"},
			args:     []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			os.Args = tt.osArgs
			result := filterArgsBeforeDash(tt.args)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStdioCommand(t *testing.T) {
	tests := []struct {
		name        string
		osArgs      []string
		expected    []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "no dash dash",
			osArgs:      []string{"hub", "--stdio", "list"},
			expectError: true,
			errorMsg:    "--stdio requires -- followed by a command",
		},
		{
			name:        "with dash dash but no command",
			osArgs:      []string{"hub", "--stdio", "list", "--"},
			expectError: true,
			errorMsg:    "--stdio requires a command after --",
		},
		{
			name:        "valid stdio command",
			osArgs:      []string{"hub", "--stdio", "list", "--", "npx", "@mcp/server"},
			expected:    []string{"npx", "@mcp/server"},
			expectError: false,
		},
		{
			name:        "single command",
			osArgs:      []string{"hub", "--stdio", "list", "--", "npx"},
			expected:    []string{"npx"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original os.Args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			os.Args = tt.osArgs
			result, err := getStdioCommand()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
