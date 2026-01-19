package js

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	_ "github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/eventloop"
	_ "github.com/dop251/goja_nodejs/process"
	_ "github.com/dop251/goja_nodejs/url"
	_ "github.com/dop251/goja_nodejs/util"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

const (
	// DefaultTimeout is the default timeout for JS execution
	DefaultTimeout = 15 * time.Second
	// MaxScriptSize is the maximum script size in bytes
	MaxScriptSize = 100 * 1024 // 100KB
	// MaxLogEntries is the maximum number of log entries allowed
	MaxLogEntries = 1000
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
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// ToolCaller abstracts tool calling for different client types
type ToolCaller interface {
	// CallTool calls a tool with the given serverID, toolName, and parameters
	// For single-server clients, serverID can be ignored or used as a default
	CallTool(ctx context.Context, serverID, toolName string, params map[string]any) (*mcp.CallToolResult, error)
}

// SessionGetter abstracts getting a client session by server ID (implemented by client.Manager)
type SessionGetter interface {
	GetClient(serverID string) (*mcp.ClientSession, error)
}

// ManagerCaller adapts a SessionGetter (like client.Manager) to the ToolCaller interface
type ManagerCaller struct {
	getter SessionGetter
}

// NewManagerCaller creates a new ManagerCaller from a SessionGetter
func NewManagerCaller(getter SessionGetter) *ManagerCaller {
	return &ManagerCaller{getter: getter}
}

// CallTool implements ToolCaller for ManagerCaller
func (m *ManagerCaller) CallTool(ctx context.Context, serverID, toolName string, params map[string]any) (*mcp.CallToolResult, error) {
	session, err := m.getter.GetClient(serverID)
	if err != nil {
		return nil, fmt.Errorf("server '%s' not found", serverID)
	}

	toolParams := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: params,
	}

	return session.CallTool(ctx, toolParams)
}

// Runtime represents a JavaScript runtime for executing tool scripts
type Runtime struct {
	logger       *zap.Logger
	caller       ToolCaller
	timeout      time.Duration
	allowedTools map[string][]string // nil = allow all
}

// Config holds runtime configuration
type Config struct {
	Timeout      time.Duration
	AllowedTools map[string][]string // map[serverID][]toolNames, nil = allow all
}

// NewRuntime creates a new JavaScript runtime
func NewRuntime(logger *zap.Logger, caller ToolCaller, cfg *Config) *Runtime {
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
		caller:       caller,
		timeout:      timeout,
		allowedTools: allowedTools,
	}
}

// Execute executes a JavaScript script with sync-only enforcement
func (r *Runtime) Execute(ctx context.Context, script string) (any, []LogEntry, error) {
	// Validate script size
	if len(script) > MaxScriptSize {
		return nil, nil, &RuntimeError{
			Type:    ErrorTypeValidation,
			Message: fmt.Sprintf("script exceeds maximum size of %d bytes", MaxScriptSize),
		}
	}

	// Apply timeout
	execCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Execute script with a Node-like event loop
	loop := eventloop.NewEventLoop()
	loop.Start()

	var (
		logs      []LogEntry
		logsMu    sync.Mutex
		result    any
		runErr    error
		vmPtr     *goja.Runtime
		vmReady   = make(chan struct{})
		resultCh  = make(chan struct{})
		readyOnce sync.Once
	)

	stopLoop := sync.Once{}
	defer stopLoop.Do(func() { loop.Stop() })

	signalReady := func() {
		readyOnce.Do(func() {
			close(resultCh)
		})
	}

	loop.RunOnLoop(func(vm *goja.Runtime) {
		vmPtr = vm
		close(vmReady)

		if err := r.injectMCPHelpers(execCtx, vm, &logs, &logsMu); err != nil {
			runErr = err
			signalReady()
			return
		}

		defer func() {
			if caught := recover(); caught != nil {
				if interrupted, ok := caught.(*goja.InterruptedError); ok {
					runErr = fmt.Errorf("execution interrupted: %v", interrupted)
				} else if val, ok := caught.(goja.Value); ok {
					runErr = fmt.Errorf("%v", val)
				} else {
					runErr = fmt.Errorf("runtime error: %v", caught)
				}
				signalReady()
			}
		}()

		res, err := vm.RunString(script)
		if err != nil {
			runErr = err
			signalReady()
			return
		}

		// Promise handling: hook into then to capture result asynchronously
		if promise, ok := res.Export().(*goja.Promise); ok && promise.State() == goja.PromiseStatePending {
			thenVal := res.ToObject(vm).Get("then")
			if thenFunc, ok := goja.AssertFunction(thenVal); ok {
				resolved := false
				resolve := func(call goja.FunctionCall) goja.Value {
					if resolved {
						return goja.Undefined()
					}
					resolved = true
					result = call.Argument(0).Export()
					signalReady()
					return goja.Undefined()
				}
				reject := func(call goja.FunctionCall) goja.Value {
					if resolved {
						return goja.Undefined()
					}
					resolved = true
					runErr = fmt.Errorf("%v", call.Argument(0))
					signalReady()
					return goja.Undefined()
				}
				thenFunc(res, vm.ToValue(resolve), vm.ToValue(reject))
				return
			}
		}

		// Settled promise or plain value
		if promise, ok := res.Export().(*goja.Promise); ok {
			switch promise.State() {
			case goja.PromiseStateFulfilled:
				result = promise.Result().Export()
			case goja.PromiseStateRejected:
				runErr = fmt.Errorf("%v", promise.Result())
			default:
				result = res.Export()
			}
		} else {
			result = res.Export()
		}

		signalReady()
	})

	// Monitor for timeout/cancellation and interrupt the VM if needed
	go func() {
		select {
		case <-execCtx.Done():
			<-vmReady
			if vmPtr != nil {
				vmPtr.Interrupt(fmt.Sprintf("execution interrupted: %v", execCtx.Err()))
			}
		case <-resultCh:
		}
	}()

	select {
	case <-resultCh:
	case <-execCtx.Done():
		<-resultCh
	}

	if runErr != nil {
		return nil, logs, r.mapError(runErr)
	}

	if execCtx.Err() == context.DeadlineExceeded {
		return nil, logs, &RuntimeError{
			Type:    ErrorTypeTimeout,
			Message: fmt.Sprintf("script execution exceeded timeout of %v", r.timeout),
		}
	}

	if execCtx.Err() != nil {
		return nil, logs, &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: "script execution cancelled",
		}
	}

	return result, logs, nil
}

// injectMCPHelpers wires mcp helpers and console log capture into the VM
func (r *Runtime) injectMCPHelpers(ctx context.Context, vm *goja.Runtime, logs *[]LogEntry, logsMu *sync.Mutex) error {
	// Store logs
	// Setup mcp helpers
	mcpObj := vm.NewObject()

	// mcp.callTool(toolName, params) - toolName format: "serverID__toolName"
	if err := mcpObj.Set("callTool", func(call goja.FunctionCall) goja.Value {
		// Check context cancellation
		select {
		case <-ctx.Done():
			panic(vm.NewGoError(fmt.Errorf("execution cancelled")))
		default:
		}

		if len(call.Arguments) != 2 {
			panic(vm.NewTypeError("mcp.callTool requires 2 arguments: toolName (e.g., 'server__tool'), params"))
		}

		fullToolName := call.Argument(0).String()
		params := call.Argument(1).Export()

		// Parse serverID__toolName format
		before, after, ok := strings.Cut(fullToolName, "__")
		if !ok {
			panic(vm.NewTypeError("toolName must be in format 'serverID__toolName' (e.g., 'github__createIssue')"))
		}

		serverID := before
		toolName := after // +2 to skip "__"

		// Call the tool
		result, err := r.callTool(ctx, serverID, toolName, params)
		if err != nil {
			panic(vm.NewGoError(err))
		}

		return vm.ToValue(result)
	}); err != nil {
		return &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to setup callTool: %v", err),
		}
	}

	// mcp.log(level, message, fields?)
	if err := mcpObj.Set("log", func(call goja.FunctionCall) goja.Value {
		logsMu.Lock()
		defer logsMu.Unlock()

		// Enforce max log entries
		if len(*logs) >= MaxLogEntries {
			return goja.Undefined()
		}

		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("mcp.log requires at least 2 arguments: level, message"))
		}

		level := call.Argument(0).String()
		message := call.Argument(1).String()

		// Sanitize log message
		message = sanitizeLogMessage(message)

		var fields map[string]any

		if len(call.Arguments) > 2 && !goja.IsUndefined(call.Argument(2)) && !goja.IsNull(call.Argument(2)) {
			exported := call.Argument(2).Export()
			if f, ok := exported.(map[string]any); ok {
				// Sanitize field values
				fields = sanitizeLogFields(f)
			}
		}

		// Validate log level
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[level] {
			level = "info"
		}

		*logs = append(*logs, LogEntry{
			Level:   level,
			Message: message,
			Fields:  fields,
		})

		return goja.Undefined()
	}); err != nil {
		return &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to setup log: %v", err),
		}
	}

	// Set mcp object in global scope
	if err := vm.Set("mcp", mcpObj); err != nil {
		return &RuntimeError{
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

			if len(*logs) >= MaxLogEntries {
				return goja.Undefined()
			}

			// Convert all arguments to strings and join them
			var parts []string
			for _, arg := range call.Arguments {
				parts = append(parts, fmt.Sprintf("%v", arg.Export()))
			}
			message := strings.Join(parts, " ")
			message = sanitizeLogMessage(message)

			*logs = append(*logs, LogEntry{
				Level:   level,
				Message: message,
			})

			return goja.Undefined()
		}
	}

	if err := consoleObj.Set("log", logFunc("info")); err != nil {
		return &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.log"}
	}
	if err := consoleObj.Set("info", logFunc("info")); err != nil {
		return &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.info"}
	}
	if err := consoleObj.Set("warn", logFunc("warn")); err != nil {
		return &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.warn"}
	}
	if err := consoleObj.Set("error", logFunc("error")); err != nil {
		return &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.error"}
	}
	if err := consoleObj.Set("debug", logFunc("debug")); err != nil {
		return &RuntimeError{Type: ErrorTypeRuntime, Message: "failed to setup console.debug"}
	}

	if err := vm.Set("console", consoleObj); err != nil {
		return &RuntimeError{
			Type:    ErrorTypeRuntime,
			Message: fmt.Sprintf("failed to set console global: %v", err),
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
func sanitizeLogFields(fields map[string]any) map[string]any {
	sanitized := make(map[string]any)

	for k, v := range fields {
		// Sanitize key
		k = sanitizeLogMessage(k)

		// Sanitize value based on type
		switch val := v.(type) {
		case string:
			sanitized[k] = sanitizeLogMessage(val)
		case map[string]any:
			sanitized[k] = sanitizeLogFields(val)
		default:
			sanitized[k] = v
		}
	}

	return sanitized
}

// callTool calls a proxied MCP tool
func (r *Runtime) callTool(ctx context.Context, serverID, toolName string, params any) (any, error) {
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
	var paramsMap map[string]any
	if params != nil {
		var ok bool
		paramsMap, ok = params.(map[string]any)
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

	// Call tool via the ToolCaller interface
	result, err := r.caller.CallTool(ctx, serverID, toolName, paramsMap)
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
		var jsonResult any
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
	return slices.Contains(slice, str)
}
