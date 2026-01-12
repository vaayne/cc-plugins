package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestRenderListDescription_InsertsToolsWithTruncationAndSorting(t *testing.T) {
	base := "Intro\n\n## Avaliable Tools\n\n" + availableToolsPlaceholder + "\n\nFooter\n"
	remote := map[string]*mcp.Tool{
		"b__two": {Name: "two", Description: strings.Repeat("word ", 55)},
		"a__one": {Name: "one", Description: "short description"},
	}

	rendered := RenderListDescription(base, remote)

	assert.Contains(t, rendered, "## Avaliable Tools")
	assert.Contains(t, rendered, "- a__one: short description")

	// Truncated to 50 words with ellipsis.
	assert.Contains(t, rendered, "- b__two: ")
	assert.Contains(t, rendered, "…")

	// Sorted by tool name.
	assert.Less(t,
		strings.Index(rendered, "- a__one:"),
		strings.Index(rendered, "- b__two:"),
	)
}

func TestRenderListDescription_NoToolsShowsNone(t *testing.T) {
	base := "Intro\n\n## Avaliable Tools\n\n" + availableToolsPlaceholder + "\n"
	rendered := RenderListDescription(base, map[string]*mcp.Tool{})
	assert.Contains(t, rendered, "- (none)")
}
