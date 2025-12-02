package js

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"mcp-hub-go/internal/client"

	"github.com/dop251/goja"
	"github.com/dop251/goja/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const (
	// DefaultTimeout is the default timeout for JS execution
	DefaultTimeout = 15 * time.Second
	// MaxScriptSize is the maximum script size in bytes
	MaxScriptSize = 100 * 1024 // 100KB
	// MaxMemoryBytes is the maximum memory allowed for VM (50MB)
	MaxMemoryBytes = 50 * 1024 * 1024
	// MaxLogEntries is the maximum number of log entries allowed
	MaxLogEntries = 1000
	// MemoryCheckInterval is how often to check memory usage
	MemoryCheckInterval = 100 * time.Millisecond
)

// ErrorType represents the type of runtime error
type ErrorType string

const (
	ErrorTypeTimeout    ErrorType = "timeout"
	ErrorTypeSyntax     ErrorType = "syntax_error"
	ErrorTypeRuntime    ErrorType = "runtime_error"
	ErrorTypeValidation ErrorType = "validation_error"
	ErrorTypeAsync      ErrorType = "async_not_allowed"
)

// RuntimeError represents a structured runtime error
type RuntimeError struct {
	Type    ErrorType `json:"type"`
	Message string    `json:"message"`
}

// Error implements the error interface
func (e *RuntimeError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// LogEntry represents a log entry from mcp.log()
type LogEntry struct {
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Runtime represents a JavaScript runtime for executing tool scripts
type Runtime struct {
	logger       *zap.Logger
	manager      *client.Manager
	timeout      time.Duration
	allowedTools map[string][]string // nil = allow all
}

// Config holds runtime configuration
type Config struct {
	Timeout      time.Duration
	AllowedTools map[string][]string // map[serverID][]toolNames, nil = allow all
}

// NewRuntime creates a new JavaScript runtime
func NewRuntime(logger *zap.Logger, manager *client.Manager, cfg *Config) *Runtime {
	timeout := DefaultTimeout
	var allowedTools map[string][]string

	if cfg != nil {
		if cfg.Timeout > 0 {
			timeout = cfg.Timeout
		}
		allowedTools = cfg.AllowedTools
	}

	return &Runtime{
		logger:       logger,
		manager:      manager,
		timeout:      timeout,
		allowedTools: allowedTools,
	}
}

// Execute executes a JavaScript script with sync-only enforcement
func (r *Runtime) Execute(ctx context.Context, script string) (interface{}, []LogEntry, error) {
	// Validate script size
	if len(script) > MaxScriptSize {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("script exceeds maximum size of %d bytes", MaxScriptSize),
		}
	}

	// Validate script doesn't contain async constructs
	if err := r.validateSyncOnly(script); err != nil {
		return nil, nil, err
	}

	// Apply timeout
	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Create VM for this execution
	vm := goja.New()

	// Execute with timeout and proper interruption
	resultChan := make(chan struct {
		result interface{}
		logs   []LogEntry
		err    error
	}, 1)

	go func() {
		result, logs, err := r.executeScript(execCtx, vm, script)
		resultChan <- struct {
			result interface{}
			logs   []LogEntry
			err    error
		}{result, logs, err}
	}()

	select {
	case res := <-resultChan:
		return res.result, res.logs, res.err
	case <-execCtx.Done():
		// Forcefully interrupt the VM on timeout or cancellation
		vm.Interrupt("execution interrupted")

		// Wait a bit for the goroutine to finish
		select {
		case res := <-resultChan:
			return res.result, res.logs, res.err
		case <-time.After(100 * time.Millisecond):
			// Goroutine still running, return timeout error
			if execCtx.Err() == context.DeadlineExceeded {
				return nil, nil, &RuntimeError{
					Type:    ErrorTypeTimeout,
					Message: fmt.Sprintf("script execution exceeded timeout of %v", r.timeout),
				}
			}
			return nil, nil, &RuntimeError{
				Type:    ErrorTypeRuntime,
				Message: "script execution cancelled",
			}
		}
	}
}

// validateSyncOnly checks if script contains async constructs using AST parsing
func (r *Runtime) validateSyncOnly(script string) error {
	// Parse the script into AST
	_, err := parser.ParseFile(nil, "", script, 0)
	if err != nil {
		// If we can't parse it, let the VM handle the syntax error
		return nil
	}

	// Check for async patterns that can bypass string detection
	// These checks are defense in depth alongside AST parsing

	// Check for Promise usage (including bracket notation like window['Promise'])
	promisePatterns := []string{
		"Promise",
		"['Promise']",
		`["Promise"]`,
		"[`Promise`]",
	}
	for _, pattern := range promisePatterns {
		if strings.Contains(script, pattern) {
			return &RuntimeError{
				Type:    ErrorTypeAsync,
				Message: "Promise usage is not allowed - only synchronous code is supported",
			}
		}
	}

	// Check for async/await keywords (including in comments and strings)
	// Remove comments and strings first
	cleanedScript := removeCommentsAndStrings(script)

	asyncPatterns := []string{
		"async ",
		"async(",
		"async\t",
		"async\n",
		"async*",
		"async\r",
	}
	for _, pattern := range asyncPatterns {
		if strings.Contains(cleanedScript, pattern) {
			return &RuntimeError{
				Type:    ErrorTypeAsync,
				Message: "async functions are not allowed - only synchronous code is supported",
			}
		}
	}

	if strings.Contains(cleanedScript, "await ") || strings.Contains(cleanedScript, "await\t") ||
		strings.Contains(cleanedScript, "await\n") || strings.Contains(cleanedScript, "await(") {
		return &RuntimeError{
			Type:    ErrorTypeAsync,
			Message: "await keyword is not allowed - only synchronous code is supported",
		}
	}

	// Check for setTimeout, setInterval, setImmediate (including bracket notation)
	asyncFuncs := []string{
		"setTimeout", "setInterval", "setImmediate",
		"['setTimeout']", "['setInterval']", "['setImmediate']",
		`["setTimeout"]`, `["setInterval"]`, `["setImmediate"]`,
	}
	for _, fn := range asyncFuncs {
		if strings.Contains(script, fn) {
			return &RuntimeError{
				Type:    ErrorTypeAsync,
				Message: fmt.Sprintf("%s is not allowed - only synchronous code is supported", fn),
			}
		}
	}

	return nil
}

// removeCommentsAndStrings removes comments and string literals from script
func removeCommentsAndStrings(script string) string {
	// Remove single-line comments
	re := regexp.MustCompile(`//.*`)
	script = re.ReplaceAllString(script, "")

	// Remove multi-line comments
	re = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	script = re.ReplaceAllString(script, "")

	// Remove string literals (simple approach - doesn't handle all edge cases but good enough)
	re = regexp.MustCompile(`"[^"\\]*(\\.[^"\\]*)*"`)
	script = re.ReplaceAllString(script, `""`)
	re = regexp.MustCompile(`'[^'\\]*(\\.[^'\\]*)*'`)
	script = re.ReplaceAllString(script, `''`)
	re = regexp.MustCompile("`[^`]*`")
	script = re.ReplaceAllString(script, "``")

	return script
}

// executeScript executes the script in the provided VM instance
func (r *Runtime) executeScript(ctx context.Context, vm *goja.Runtime, script string) (interface{}, []LogEntry, error) {
	// Store logs
	var logs []LogEntry
	var logsMu sync.Mutex

	// Start context monitor that will interrupt VM on cancellation
	stopMonitor := make(chan bool, 1)
	defer func() {
		stopMonitor <- true
	}()

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopMonitor:
				return
			case <-ctx.Done():
				// Interrupt the VM when context is cancelled
				vm.Interrupt("context cancelled")
				return
			case <-ticker.C:
				// Periodic check for context cancellation
				select {
				case <-ctx.Done():
					vm.Interrupt("context cancelled")
					return
				default:
				}
			}
		}
	}()

	// Setup mcp helpers
	mcpObj := vm.NewObject()

	// mcp.callTool(toolName, params) - toolName format: "serverID.toolName"
	if err := mcpObj.Set("callTool", func(call goja.FunctionCall) goja.Value {
		// Check context cancellation
		select {
		case <-ctx.Done():
			panic(vm.NewGoError(fmt.Errorf("execution cancelled")))
		default:
		}

		if len(call.Arguments) != 2 {
			panic(vm.NewTypeError("mcp.callTool requires 2 arguments: toolName (e.g., 'server.tool'), params"))
		}

		fullToolName := call.Argument(0).String()
		params := call.Argument(1).Export()

		// Parse serverID.toolName format
		dotIndex := strings.Index(fullToolName, ".")
		if dotIndex == -1 {
			panic(vm.NewTypeError("toolName must be in format 'serverID.toolName' (e.g., 'grep.searchGitHub')"))
		}

		serverID := fullToolName[:dotIndex]
		toolName := fullToolName[dotIndex+1:]

		// Call the tool
		result, err := r.callTool(ctx, serverID, toolName, params)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return vm.ToValue(result)
	}); err != nil {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to setup callTool: %v", err),
		}
	}

	// mcp.log(level, message, fields?)
	if err := mcpObj.Set("log", func(call goja.FunctionCall) goja.Value {
		logsMu.Lock()
		defer logsMu.Unlock()

		// Enforce max log entries
		if len(logs) >= MaxLogEntries {
			return goja.Undefined()
		}

		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("mcp.log requires at least 2 arguments: level, message"))
		}

		level := call.Argument(0).String()
		message := call.Argument(1).String()

		// Sanitize log message
		message = sanitizeLogMessage(message)

		var fields map[string]interface{}

		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Argument(2)) && !goja.IsNull(call.Argument(2)) {
			exported := call.Argument(2).Export()
			if f, ok := exported.(map[string]interface{}); ok {
				// Sanitize field values
				fields = sanitizeLogFields(f)
			}
		}

		// Validate log level
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[level] {
			level = "info"
		}

		logs = append(logs, LogEntry{
			Level:   level,
			Message: message,
			Fields:  fields,
		})

		return goja.Undefined()
	}); err != nil {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to setup log: %v", err),
		}
	}

	// Set mcp object in global scope
	if err := vm.Set("mcp", mcpObj); err != nil {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to set mcp global: %v", err),
		}
	}

	// Add console object as an alias for mcp.log (LLMs naturally use console.log)
	consoleObj := vm.NewObject()
	logFunc := func(level string) func(call goja.FunctionCall) goja.Value {
		return func(call goja.FunctionCall) goja.Value {
			logsMu.Lock()
			defer logsMu.Unlock()

			if len(logs) >= MaxLogEntries {
				return goja.Undefined()
			}

			// Convert all arguments to strings and join them
			var parts []string
			for _, arg := range call.Arguments {
				parts = append(parts, fmt.Sprintf("%v", arg.Export()))
			}
			message := strings.Join(parts, " ")
			message = sanitizeLogMessage(message)

			logs = append(logs, LogEntry{
				Level:   level,
				Message: message,
			})

			return goja.Undefined()
		}
	}

	if err := consoleObj.Set("log", logFunc("info")); err != nil {
		return nil, nil, &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.log"}
	}
	if err := consoleObj.Set("info", logFunc("info")); err != nil {
		return nil, nil, &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.info"}
	}
	if err := consoleObj.Set("warn", logFunc("warn")); err != nil {
		return nil, nil, &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.warn"}
	}
	if err := consoleObj.Set("error", logFunc("error")); err != nil {
		return nil, nil, &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.error"}
	}
	if err := consoleObj.Set("debug", logFunc("debug")); err != nil {
		return nil, nil, &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.debug"}
	}

	if err := vm.Set("console", consoleObj); err != nil {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to set console global: %v", err),
		}
	}

	// Block dangerous globals and freeze prototypes
	if err := blockDangerousGlobals(vm); err != nil {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to secure globals: %v", err),
		}
	}

	// Execute script
	var result goja.Value
	var execErr error

	func() {
		defer func() {
			if caught := recover(); caught != nil {
				if gojaErr, ok := caught.(*goja.InterruptedError); ok {
					execErr = fmt.Errorf("execution interrupted: %v", gojaErr)
				} else {
					execErr = fmt.Errorf("runtime error: %v", caught)
				}
			}
		}()

		// Check context before execution
		select {
		case <-ctx.Done():
			execErr = ctx.Err()
			return
		default:
		}

		result, execErr = vm.RunString(script)
	}()

	if execErr != nil {
		return nil, logs, r.mapError(execErr)
	}

	// Export result
	exported := result.Export()
	return exported, logs, nil
}

// blockDangerousGlobals blocks access to dangerous global objects and freezes prototypes
func blockDangerousGlobals(vm *goja.Runtime) error {
	// IMPORTANT: Freeze prototypes and block constructors BEFORE setting globals to undefined
	// Otherwise we can't access Function.prototype to freeze it
	_, err := vm.RunString(`
		(function() {
			'use strict';

			// Block access to constructor chains more comprehensively
			// This prevents access via (function(){}).constructor
			var blockConstructor = function() {
				throw new Error('Access to constructor is not allowed');
			};

			// Freeze built-in prototypes and block their constructor property
			if (typeof Object !== 'undefined' && Object.prototype) {
				try {
					Object.defineProperty(Object.prototype, 'constructor', {
						get: blockConstructor,
						set: blockConstructor,
						configurable: false
					});
				} catch (e) {
					// Already defined, try to override
				}
				Object.freeze(Object.prototype);
			}
			if (typeof Array !== 'undefined' && Array.prototype) {
				try {
					Object.defineProperty(Array.prototype, 'constructor', {
						get: blockConstructor,
						set: blockConstructor,
						configurable: false
					});
				} catch (e) {}
				Object.freeze(Array.prototype);
			}
			if (typeof String !== 'undefined' && String.prototype) {
				try {
					Object.defineProperty(String.prototype, 'constructor', {
						get: blockConstructor,
						set: blockConstructor,
						configurable: false
					});
				} catch (e) {}
				Object.freeze(String.prototype);
			}
			if (typeof Number !== 'undefined' && Number.prototype) {
				try {
					Object.defineProperty(Number.prototype, 'constructor', {
						get: blockConstructor,
						set: blockConstructor,
						configurable: false
					});
				} catch (e) {}
				Object.freeze(Number.prototype);
			}
			if (typeof Boolean !== 'undefined' && Boolean.prototype) {
				try {
					Object.defineProperty(Boolean.prototype, 'constructor', {
						get: blockConstructor,
						set: blockConstructor,
						configurable: false
					});
				} catch (e) {}
				Object.freeze(Boolean.prototype);
			}

			// Block Function prototype constructor BEFORE we set Function to undefined
			if (typeof Function !== 'undefined' && Function.prototype) {
				try {
					Object.defineProperty(Function.prototype, 'constructor', {
						get: blockConstructor,
						set: blockConstructor,
						configurable: false
					});
				} catch (e) {}
				Object.freeze(Function.prototype);
			}
		})();
	`)

	if err != nil {
		return fmt.Errorf("failed to freeze prototypes: %w", err)
	}

	// NOW block dangerous constructors and globals after prototypes are frozen
	dangerousGlobals := []string{
		"eval",
		"Function",
		"GeneratorFunction",
		"AsyncFunction",
		"AsyncGeneratorFunction",
		"Reflect",
		"Proxy",
		"WebAssembly",
	}

	for _, name := range dangerousGlobals {
		if err := vm.Set(name, goja.Undefined()); err != nil {
			return fmt.Errorf("failed to block %s: %w", name, err)
		}
	}

	return nil
}

// sanitizeLogMessage removes control characters and limits length
func sanitizeLogMessage(msg string) string {
	// Remove ANSI escape codes
	ansiEscape := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	msg = ansiEscape.ReplaceAllString(msg, "")

	// Remove other control characters (except newlines and tabs)
	controlChars := regexp.MustCompile(`[\x00-\x08\x0B-\x0C\x0E-\x1F\x7F]`)
	msg = controlChars.ReplaceAllString(msg, "")

	// Limit message length
	const maxMessageLength = 10000
	if len(msg) > maxMessageLength {
		msg = msg[:maxMessageLength] + "..."
	}

	return msg
}

// sanitizeLogFields sanitizes all field values in a map
func sanitizeLogFields(fields map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for k, v := range fields {
		// Sanitize key
		k = sanitizeLogMessage(k)

		// Sanitize value based on type
		switch val := v.(type) {
		case string:
			sanitized[k] = sanitizeLogMessage(val)
		case map[string]interface{}:
			sanitized[k] = sanitizeLogFields(val)
		default:
			sanitized[k] = v
		}
	}

	return sanitized
}

// callTool calls a proxied MCP tool
func (r *Runtime) callTool(ctx context.Context, serverID, toolName string, params interface{}) (interface{}, error) {
	fullToolName := serverID + "." + toolName

	// Validate inputs
	if serverID == "" {
		return nil, fmt.Errorf("serverID is required in tool name")
	}
	if toolName == "" {
		return nil, fmt.Errorf("toolName is required after the dot")
	}

	// Convert params to map for CallToolParams - do this BEFORE authorization/client checks
	// so we get proper type error messages
	var paramsMap map[string]interface{}
	if params != nil {
		var ok bool
		paramsMap, ok = params.(map[string]interface{})
		if !ok {
			// Proper error for type mismatch instead of silent failure
			return nil, fmt.Errorf("params must be an object, got %T", params)
		}
	}

	// Check tool authorization
	if r.allowedTools != nil {
		allowed, ok := r.allowedTools[serverID]
		if !ok || !contains(allowed, toolName) {
			return nil, fmt.Errorf("tool '%s' is not authorized", fullToolName)
		}
	}

	// Get client session
	session, err := r.manager.GetClient(serverID)
	if err != nil {
		return nil, fmt.Errorf("server '%s' not found - check server name or run 'search' to find available tools", serverID)
	}

	// Call tool
	toolParams := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: paramsMap,
	}

	result, err := session.CallTool(ctx, toolParams)
	if err != nil {
		// Provide helpful error message with sanitized details
		errMsg := sanitizeToolError(err)
		return nil, fmt.Errorf("tool '%s' failed: %s", fullToolName, errMsg)
	}

	// Extract result from content
	if len(result.Content) == 0 {
		return nil, nil
	}

	// Return first content item
	switch content := result.Content[0].(type) {
	case *mcp.TextContent:
		// Try to parse as JSON, otherwise return as string
		var jsonResult interface{}
		if err := json.Unmarshal([]byte(content.Text), &jsonResult); err == nil {
			return jsonResult, nil
		}
		return content.Text, nil
	default:
		return nil, fmt.Errorf("unsupported content type from '%s'", fullToolName)
	}
}

// sanitizeToolError extracts useful error info while removing sensitive details
func sanitizeToolError(err error) string {
	if err == nil {
		return "unknown error"
	}

	errStr := err.Error()

	// Check for common error patterns and provide helpful messages
	switch {
	case strings.Contains(errStr, "not found"):
		return "tool not found"
	case strings.Contains(errStr, "connection refused"):
		return "server connection refused"
	case strings.Contains(errStr, "timeout"):
		return "request timeout"
	case strings.Contains(errStr, "context canceled"):
		return "request canceled"
	case strings.Contains(errStr, "invalid argument"):
		return "invalid arguments"
	case strings.Contains(errStr, "permission denied"):
		return "permission denied"
	default:
		// For other errors, return a sanitized version
		// Remove file paths and other sensitive info
		sanitized := errStr
		// Remove common path patterns
		for _, prefix := range []string{"/Users/", "/home/", "C:\\", "/var/", "/tmp/"} {
			if idx := strings.Index(sanitized, prefix); idx != -1 {
				// Find the end of the path (space or end of string)
				endIdx := strings.IndexAny(sanitized[idx:], " \t\n:\"'")
				if endIdx == -1 {
					sanitized = sanitized[:idx] + "[path]"
				} else {
					sanitized = sanitized[:idx] + "[path]" + sanitized[idx+endIdx:]
				}
			}
		}
		// Truncate if too long
		if len(sanitized) > 100 {
			sanitized = sanitized[:100] + "..."
		}
		return sanitized
	}
}

// mapError maps VM errors to RuntimeError
func (r *Runtime) mapError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	// Check for interruption (timeout/cancellation)
	if strings.Contains(errMsg, "execution interrupted") || strings.Contains(errMsg, "Interrupt") {
		return &RuntimeError{
			Type:    ErrorTypeTimeout,
			Message: sanitizeError(errMsg),
		}
	}

	// Check for syntax errors
	if strings.Contains(errMsg, "SyntaxError") || strings.Contains(errMsg, "Line ") {
		return &RuntimeError{
			Type:    ErrorTypeSyntax,
			Message: sanitizeError(errMsg),
		}
	}

	// Check for type errors
	if strings.Contains(errMsg, "TypeError") {
		return &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: sanitizeError(errMsg),
		}
	}

	// Check for reference errors
	if strings.Contains(errMsg, "ReferenceError") {
		return &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: sanitizeError(errMsg),
		}
	}

	// Generic runtime error
	return &RuntimeError{
		Type:    ErrorTypeRuntime,
		Message: sanitizeError(errMsg),
	}
}

// sanitizeError removes potentially sensitive information from error messages
func sanitizeError(msg string) string {
	// Remove file paths that might leak system information
	msg = strings.ReplaceAll(msg, "\r\n", " ")
	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, "\t", " ")

	// Limit error message length
	const maxErrorLength = 500
	if len(msg) > maxErrorLength {
		msg = msg[:maxErrorLength] + "..."
	}

	return msg
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
