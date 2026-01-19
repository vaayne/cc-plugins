package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"hub/internal/logging"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// createRemoteClient creates a RemoteClient from command flags
func createRemoteClient(ctx context.Context, cmd *cobra.Command) (*RemoteClient, error) {
	url, _ := cmd.Flags().GetString("url")
	transport, _ := cmd.Flags().GetString("transport")
	timeout, _ := cmd.Flags().GetInt("timeout")
	headers, _ := cmd.Flags().GetStringArray("header")
	verbose, _ := cmd.Flags().GetBool("verbose")
	logFile, _ := cmd.Flags().GetString("log-file")

	// Default to http for remote commands
	if transport == "" {
		transport = "http"
	}

	// Initialize logging
	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return nil, err
	}

	opts := RemoteClientOpts{
		ServerURL: url,
		Transport: transport,
		Headers:   parseHeaders(headers),
		Timeout:   timeout,
		Logger:    logging.Logger,
	}

	return NewRemoteClient(ctx, opts)
}

func createConfigClient(ctx context.Context, cmd *cobra.Command) (*ConfigClient, error) {
	configPath, _ := cmd.Flags().GetString("config")
	timeout, _ := cmd.Flags().GetInt("timeout")
	logger := getLogger(cmd)

	return NewConfigClient(ctx, configPath, logger, time.Duration(timeout)*time.Second)
}

// parseHeaders parses headers from []string in format "Key: Value" into map[string]string.
// Malformed headers (without ":") are silently skipped.
// Header values can contain environment variables that are expanded (e.g., $TOKEN or ${TOKEN}).
func parseHeaders(headers []string) map[string]string {
	result := make(map[string]string)
	for _, h := range headers {
		parts := strings.SplitN(h, ":", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
		// Malformed headers without ":" are silently skipped
	}
	return result
}

// getLogger returns a configured logger based on command flags
func getLogger(cmd *cobra.Command) *zap.Logger {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logFile, _ := cmd.Flags().GetString("log-file")

	logLevel := zapcore.InfoLevel
	if verbose {
		logLevel = zapcore.DebugLevel
	}

	logConfig := logging.Config{
		LogLevel:    logLevel,
		LogFilePath: logFile,
	}
	if _, err := logging.InitLogger(logConfig); err != nil {
		return zap.NewNop() // Safe fallback
	}

	return logging.Logger
}

// ToolNameMapper maintains bidirectional mapping between original and JS method names
type ToolNameMapper struct {
	toJS       map[string]string // original -> jsName
	toOriginal map[string]string // jsName -> original
}

// NewToolNameMapper creates a mapper from a list of tools
func NewToolNameMapper(tools []*mcp.Tool) *ToolNameMapper {
	m := &ToolNameMapper{
		toJS:       make(map[string]string),
		toOriginal: make(map[string]string),
	}
	for _, tool := range tools {
		jsName := toJSMethodName(tool.Name)
		m.toJS[tool.Name] = jsName
		m.toOriginal[jsName] = tool.Name
	}
	return m
}

func NewToolNameMapperWithCollisionCheck(tools []*mcp.Tool) (*ToolNameMapper, error) {
	collisions := make(map[string][]string)
	for _, tool := range tools {
		jsName := toJSMethodName(tool.Name)
		collisions[jsName] = append(collisions[jsName], tool.Name)
	}

	var parts []string
	for jsName, originals := range collisions {
		if len(originals) > 1 {
			parts = append(parts, fmt.Sprintf("%s: %s", jsName, strings.Join(originals, ", ")))
		}
	}

	if len(parts) > 0 {
		return nil, fmt.Errorf("tool name mapping collision: %s", strings.Join(parts, "; "))
	}

	return NewToolNameMapper(tools), nil
}

// ToJSName converts an original tool name to its JS method name
func (m *ToolNameMapper) ToJSName(original string) string {
	if jsName, ok := m.toJS[original]; ok {
		return jsName
	}
	return toJSMethodName(original)
}

// ToOriginal converts a JS method name back to its original tool name
// Returns the input unchanged if not found (allows pass-through)
func (m *ToolNameMapper) ToOriginal(jsName string) string {
	if original, ok := m.toOriginal[jsName]; ok {
		return original
	}
	return jsName
}

func ensureNamespacedToolName(name string) error {
	if !strings.Contains(name, "__") {
		return fmt.Errorf("tool name must include server prefix (server__tool) when using --config")
	}
	return nil
}

// toJSMethodName converts a tool name to a valid JS method name (camelCase)
// Examples:
//   - get_code_context_exa -> getCodeContextExa
//   - web_search_exa -> webSearchExa
//   - searchGitHub -> searchGitHub (already valid)
//   - my-tool-name -> myToolName
func toJSMethodName(name string) string {
	if name == "" {
		return name
	}

	var result strings.Builder
	capitalizeNext := false
	isFirstChar := true

	for _, r := range name {
		// Treat underscores and hyphens as word separators
		if r == '_' || r == '-' {
			capitalizeNext = true
			continue
		}

		// Handle the character
		if isFirstChar {
			// First character should always be lowercase for camelCase
			result.WriteRune(unicode.ToLower(r))
			isFirstChar = false
			capitalizeNext = false
		} else if capitalizeNext && unicode.IsLetter(r) {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// getStdioCommand extracts the stdio command from os.Args after the "--" separator.
// Returns the command slice and an error if --stdio is used without a command.
func getStdioCommand() ([]string, error) {
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			if i+1 >= len(args) {
				return nil, fmt.Errorf("--stdio requires a command after --")
			}
			return args[i+1:], nil
		}
	}
	return nil, fmt.Errorf("--stdio requires -- followed by a command (e.g., hub --stdio list -- npx @mcp/server)")
}

// getStdioCommandLength returns the number of args that belong to the stdio command
// (everything after "--" in os.Args). Returns 0 if no "--" is found.
func getStdioCommandLength() int {
	for i, arg := range os.Args {
		if arg == "--" {
			return len(os.Args) - i - 1
		}
	}
	return 0
}

// filterArgsBeforeDash returns only the args that come before "--" separator.
// Since Cobra strips "--" from the args it passes to commands, we need to
// calculate how many args belong to the stdio command and exclude them.
func filterArgsBeforeDash(args []string) []string {
	stdioLen := getStdioCommandLength()
	if stdioLen == 0 {
		return args
	}
	// Remove the last stdioLen args (they belong to stdio command)
	if stdioLen >= len(args) {
		return []string{}
	}
	return args[:len(args)-stdioLen]
}

// createStdioClientFromCmd creates a StdioClient using command flags and the stdio command from os.Args
func createStdioClientFromCmd(ctx context.Context, cmd *cobra.Command) (*StdioClient, error) {
	timeout, _ := cmd.Flags().GetInt("timeout")
	verbose, _ := cmd.Flags().GetBool("verbose")
	logFile, _ := cmd.Flags().GetString("log-file")

	stdioCmd, err := getStdioCommand()
	if err != nil {
		return nil, err
	}

	return createStdioClient(ctx, stdioCmd, timeout, verbose, logFile)
}
