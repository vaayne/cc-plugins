package tools

import (
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const availableToolsPlaceholder = "{{AVAILABLE_TOOLS}}"

// RenderListDescription renders the list tool's markdown description by injecting a
// summary of currently available remote tools.
func RenderListDescription(baseMarkdown string, remoteTools map[string]*mcp.Tool) string {
	lines := renderAvailableToolsLines(remoteTools)

	if strings.Contains(baseMarkdown, availableToolsPlaceholder) {
		return strings.Replace(baseMarkdown, availableToolsPlaceholder, strings.Join(lines, "\n"), 1)
	}

	// Fallback: append the section if the placeholder is missing.
	var sb strings.Builder
	sb.WriteString(strings.TrimRight(baseMarkdown, "\n"))
	sb.WriteString("\n\n## Avaliable Tools\n")
	sb.WriteString(strings.Join(lines, "\n"))
	sb.WriteString("\n")
	return sb.String()
}

func renderAvailableToolsLines(remoteTools map[string]*mcp.Tool) []string {
	if len(remoteTools) == 0 {
		return []string{"- (none)"}
	}

	toolNames := make([]string, 0, len(remoteTools))
	for name := range remoteTools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	lines := make([]string, 0, len(toolNames))
	for _, name := range toolNames {
		tool := remoteTools[name]
		desc := ""
		if tool != nil {
			desc = tool.Description
		}
		if strings.TrimSpace(desc) == "" {
			desc = name
		}
		lines = append(lines, "- "+name+": "+truncateWords(desc, 50))
	}
	return lines
}

func truncateWords(s string, maxWords int) string {
	if maxWords <= 0 {
		return ""
	}
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:maxWords], " ") + "…"
}

