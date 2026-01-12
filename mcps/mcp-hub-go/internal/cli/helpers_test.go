package cli

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestToJSMethodName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case", "get_code_context_exa", "getCodeContextExa"},
		{"snake_case_simple", "web_search_exa", "webSearchExa"},
		{"already_camelCase", "searchGitHub", "searchGitHub"},
		{"kebab-case", "my-tool-name", "myToolName"},
		{"mixed_separators", "my_tool-name", "myToolName"},
		{"leading_underscore", "_private_tool", "privateTool"},
		{"trailing_underscore", "tool_", "tool"},
		{"consecutive_underscores", "get__tool", "getTool"},
		{"single_word", "tool", "tool"},
		{"uppercase_start", "Tool", "tool"},
		{"empty_string", "", ""},
		{"numbers", "tool_v2_api", "toolV2Api"},
		{"all_caps_word", "GET_DATA", "gETDATA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toJSMethodName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToolNameMapper(t *testing.T) {
	tools := []*mcp.Tool{
		{Name: "get_code_context_exa"},
		{Name: "web_search_exa"},
		{Name: "searchGitHub"},
	}

	mapper := NewToolNameMapper(tools)

	t.Run("ToJSName", func(t *testing.T) {
		assert.Equal(t, "getCodeContextExa", mapper.ToJSName("get_code_context_exa"))
		assert.Equal(t, "webSearchExa", mapper.ToJSName("web_search_exa"))
		assert.Equal(t, "searchGitHub", mapper.ToJSName("searchGitHub"))
	})

	t.Run("ToOriginal", func(t *testing.T) {
		assert.Equal(t, "get_code_context_exa", mapper.ToOriginal("getCodeContextExa"))
		assert.Equal(t, "web_search_exa", mapper.ToOriginal("webSearchExa"))
		assert.Equal(t, "searchGitHub", mapper.ToOriginal("searchGitHub"))
	})

	t.Run("ToOriginal_passthrough", func(t *testing.T) {
		// Unknown names should pass through unchanged
		assert.Equal(t, "unknownTool", mapper.ToOriginal("unknownTool"))
	})
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]string
	}{
		{
			name:     "valid headers",
			input:    []string{"Authorization: Bearer token", "Content-Type: application/json"},
			expected: map[string]string{"Authorization": "Bearer token", "Content-Type": "application/json"},
		},
		{
			name:     "malformed header skipped",
			input:    []string{"Authorization: Bearer token", "invalid-header"},
			expected: map[string]string{"Authorization": "Bearer token"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: map[string]string{},
		},
		{
			name:     "value with colon",
			input:    []string{"Header: value:with:colons"},
			expected: map[string]string{"Header": "value:with:colons"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseHeaders(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
