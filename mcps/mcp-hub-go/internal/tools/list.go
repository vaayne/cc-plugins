package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"mcp-hub-go/internal/client"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed list_description.md
var ListDescription string

// ListToolResult represents a tool in the list (kept for internal use)
type ListToolResult struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Server      string                 `json:"server"`
	InputSchema map[string]interface{} `json:"inputSchema,omitempty"`
}

// ListToolsResponse represents the response from the list tool (kept for internal use)
type ListToolsResponse struct {
	Tools []ListToolResult `json:"tools"`
	Total int              `json:"total"`
}

// jsonSchemaTypeToJS converts JSON Schema types to JavaScript types for JSDoc
func jsonSchemaTypeToJS(schemaType interface{}) string {
	switch t := schemaType.(type) {
	case string:
		switch t {
		case "string":
			return "string"
		case "number", "integer":
			return "number"
		case "boolean":
			return "boolean"
		case "array":
			return "Array"
		case "object":
			return "Object"
		default:
			return "*"
		}
	default:
		return "*"
	}
}

// schemaToJSDoc generates a JSDoc comment and function stub from a tool's schema
func schemaToJSDoc(toolName, description string, inputSchema map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("/**\n")

	// Use tool name as fallback if description is empty
	if description == "" {
		description = toolName
	}
	sb.WriteString(fmt.Sprintf(" * %s\n", description))

	// Extract properties from schema
	if inputSchema != nil {
		if props, ok := inputSchema["properties"].(map[string]interface{}); ok && len(props) > 0 {
			sb.WriteString(" * @param {Object} params - Parameters\n")

			// Sort property names for consistent output
			propNames := make([]string, 0, len(props))
			for name := range props {
				propNames = append(propNames, name)
			}
			sort.Strings(propNames)

			for _, propName := range propNames {
				propDef := props[propName]
				propMap, ok := propDef.(map[string]interface{})
				if !ok {
					continue
				}

				jsType := jsonSchemaTypeToJS(propMap["type"])
				propDesc := ""
				if d, ok := propMap["description"].(string); ok {
					propDesc = d
				}

				// Handle enum values
				if enum, ok := propMap["enum"].([]interface{}); ok && len(enum) > 0 {
					enumStrs := make([]string, 0, len(enum))
					for _, e := range enum {
						enumStrs = append(enumStrs, fmt.Sprintf("%v", e))
					}
					if propDesc != "" {
						propDesc += " "
					}
					propDesc += fmt.Sprintf("(one of: %s)", strings.Join(enumStrs, ", "))
				}

				if propDesc == "" {
					sb.WriteString(fmt.Sprintf(" * @param {%s} params.%s\n", jsType, propName))
				} else {
					sb.WriteString(fmt.Sprintf(" * @param {%s} params.%s - %s\n", jsType, propName, propDesc))
				}
			}
		}
	}

	sb.WriteString(" */\n")
	sb.WriteString(fmt.Sprintf("function %s(params) {}\n", toolName))

	return sb.String()
}

// HandleListTool handles the list tool call
func HandleListTool(ctx context.Context, manager *client.Manager, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Parse arguments
	var args struct {
		Server string `json:"server"` // optional: filter by server name
		Query  string `json:"query"`  // optional: comma-separated keywords; tool matches if ANY keyword appears in name or description
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse list arguments: %w", err)
	}

	// Validate query length
	const maxQueryLength = 1000
	if len(args.Query) > maxQueryLength {
		return nil, fmt.Errorf("query too long (max %d characters)", maxQueryLength)
	}

	var results []ListToolResult
	const maxResults = 100 // Limit results to prevent DoS
	totalMatches := 0

	// Get all remote tools
	allRemoteTools := manager.GetAllTools()
	for namespacedName, tool := range allRemoteTools {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Extract server ID from namespaced name (format: serverID__toolName)
		separatorIndex := strings.Index(namespacedName, "__")
		serverID := "unknown"
		if separatorIndex != -1 {
			serverID = namespacedName[:separatorIndex]
		}

		// Filter by server if specified
		if args.Server != "" && !strings.EqualFold(serverID, args.Server) {
			continue
		}

		// Filter by query keywords if specified
		if !matchesKeywords(tool.Name, tool.Description, args.Query) {
			continue
		}

		totalMatches++

		if len(results) >= maxResults {
			continue
		}

		// Convert InputSchema to map if possible
		var inputSchema map[string]interface{}
		if tool.InputSchema != nil {
			if schema, ok := tool.InputSchema.(map[string]interface{}); ok {
				inputSchema = schema
			}
		}

		results = append(results, ListToolResult{
			Name:        namespacedName,
			Description: tool.Description,
			Server:      serverID,
			InputSchema: inputSchema,
		})
	}

	// Sort results by name for consistent output
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	// Build JavaScript function stubs output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("// Total: %d tools", totalMatches))
	if totalMatches > maxResults {
		output.WriteString(fmt.Sprintf(" (showing first %d)", maxResults))
	}
	output.WriteString("\n\n")

	for i, tool := range results {
		if i > 0 {
			output.WriteString("\n")
		}
		output.WriteString(schemaToJSDoc(tool.Name, tool.Description, tool.InputSchema))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: output.String(),
			},
		},
	}, nil
}

// matchesKeywords checks if tool matches any of the comma-separated keywords
func matchesKeywords(name, description, query string) bool {
	if query == "" {
		return true // no filter, match all
	}

	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	// Split by comma and match if ANY keyword appears in name or description
	keywords := strings.Split(query, ",")
	foundKeyword := false
	for _, raw := range keywords {
		kw := strings.TrimSpace(strings.ToLower(raw))
		if kw == "" {
			continue
		}
		foundKeyword = true
		if strings.Contains(nameLower, kw) || strings.Contains(descLower, kw) {
			return true
		}
	}

	// If no non-empty keywords were provided, treat as no filter
	if !foundKeyword {
		return true
	}

	return false
}
