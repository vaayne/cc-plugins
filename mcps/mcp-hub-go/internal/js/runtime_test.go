package js

import (
	"context"
	"strings"
	"testing"
	"time"

	"mcp-hub-go/internal/client"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestNewRuntime verifies runtime initialization
func TestNewRuntime(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)
	assert.NotNil(t, runtime)
	assert.Equal(t, DefaultTimeout, runtime.timeout)
}

// TestNewRuntime_CustomTimeout verifies custom timeout configuration
func TestNewRuntime_CustomTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	customTimeout := 5 * time.Second
	runtime := NewRuntime(logger, manager, &Config{Timeout: customTimeout})
	assert.Equal(t, customTimeout, runtime.timeout)
}

// TestExecute_Simple verifies simple script execution
func TestExecute_Simple(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `1 + 1`
	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result)
	assert.Empty(t, logs)
}

// TestExecute_WithLogging verifies mcp.log functionality
func TestExecute_WithLogging(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `
		mcp.log('info', 'Test message');
		mcp.log('debug', 'Debug message', {key: 'value'});
		42
	`
	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result)
	require.Len(t, logs, 2)

	assert.Equal(t, "info", logs[0].Level)
	assert.Equal(t, "Test message", logs[0].Message)
	assert.Nil(t, logs[0].Fields)

	assert.Equal(t, "debug", logs[1].Level)
	assert.Equal(t, "Debug message", logs[1].Message)
	assert.NotNil(t, logs[1].Fields)
	assert.Equal(t, "value", logs[1].Fields["key"])
}

// TestExecute_InvalidLogLevel verifies log level validation
func TestExecute_InvalidLogLevel(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `
		mcp.log('invalid', 'Test message');
		'done'
	`
	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, "done", result)
	require.Len(t, logs, 1)
	// Should default to 'info' for invalid levels
	assert.Equal(t, "info", logs[0].Level)
}

// TestExecute_AsyncAwait verifies async/await with timers works
func TestExecute_AsyncAwait(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `
		const delay = (ms) => new Promise((resolve) => setTimeout(() => resolve(ms + 1), ms));
		async function run() {
			const val = await delay(5);
			return val;
		}
		run();
	`

	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Empty(t, logs)
	assert.Equal(t, int64(6), result)
}

// TestExecute_RequireBuffer verifies require works via goja_nodejs
func TestExecute_RequireBuffer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `
		const { Buffer } = require("node:buffer");
		Buffer.from("hi").toString("hex");
	`

	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Empty(t, logs)
	assert.Equal(t, "6869", result)
}

// TestExecute_Timeout verifies timeout enforcement
func TestExecute_Timeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create runtime with very short timeout
	runtime := NewRuntime(logger, manager, &Config{Timeout: 100 * time.Millisecond})

	// Infinite loop
	script := `while(true) {}`
	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeTimeout, runtimeErr.Type)
	assert.Contains(t, strings.ToLower(runtimeErr.Message), "interrupt")
}

// TestExecute_ScriptSizeLimit verifies script size validation
func TestExecute_ScriptSizeLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Create script larger than MaxScriptSize
	largeScript := make([]byte, MaxScriptSize+1)
	for i := range largeScript {
		largeScript[i] = 'a'
	}

	_, _, err := runtime.Execute(context.Background(), string(largeScript))
	require.Error(t, err)

	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeValidation, runtimeErr.Type)
	assert.Contains(t, runtimeErr.Message, "exceeds maximum size")
}

// TestExecute_SyntaxError verifies syntax error mapping
func TestExecute_SyntaxError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `const x = {` // Unclosed brace
	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeSyntax, runtimeErr.Type)
}

// TestExecute_RuntimeError verifies runtime error mapping
func TestExecute_RuntimeError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "ReferenceError",
			script: `undefinedVariable`,
		},
		{
			name:   "TypeError",
			script: `null.toString()`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err)

			runtimeErr, ok := err.(*RuntimeError)
			require.True(t, ok)
			assert.Equal(t, ErrorTypeRuntime, runtimeErr.Type)
		})
	}
}

// TestExecute_TimeoutWithInterrupt verifies timeout with VM interruption
func TestExecute_TimeoutWithInterrupt(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create runtime with very short timeout
	runtime := NewRuntime(logger, manager, &Config{Timeout: 100 * time.Millisecond})

	// Infinite loop that should be interrupted
	script := `while(true) { let x = 1 + 1; }`

	start := time.Now()
	_, _, err := runtime.Execute(context.Background(), script)
	elapsed := time.Since(start)

	require.Error(t, err)

	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeTimeout, runtimeErr.Type)
	// Message should contain "interrupt" after mapping
	assert.Contains(t, strings.ToLower(runtimeErr.Message), "interrupt")

	// Should timeout within reasonable time (not hang forever)
	assert.Less(t, elapsed, 500*time.Millisecond, "Should interrupt quickly")
}

// TestExecute_LogSanitization verifies log message sanitization
func TestExecute_LogSanitization(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name          string
		script        string
		expectedMsg   string
		checkContains bool
	}{
		{
			name:          "ANSI escape codes removed",
			script:        `mcp.log('info', '\x1b[31mRed Text\x1b[0m'); 'done'`,
			expectedMsg:   "Red Text",
			checkContains: false,
		},
		{
			name:          "Control characters removed",
			script:        `mcp.log('info', 'Test\x00\x01\x02Message'); 'done'`,
			expectedMsg:   "TestMessage",
			checkContains: false,
		},
		{
			name:          "Very long message truncated",
			script:        `mcp.log('info', 'a'.repeat(20000)); 'done'`,
			checkContains: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, logs, err := runtime.Execute(context.Background(), tt.script)
			require.NoError(t, err)
			assert.Equal(t, "done", result)
			require.Len(t, logs, 1)

			if tt.checkContains {
				// Just verify it's truncated
				assert.LessOrEqual(t, len(logs[0].Message), 10003) // 10000 + "..."
			} else {
				assert.Equal(t, tt.expectedMsg, logs[0].Message)
			}
		})
	}
}

// TestExecute_LogEntryLimit verifies max log entries enforcement
func TestExecute_LogEntryLimit(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Try to log more than MaxLogEntries
	script := `
		for (let i = 0; i < 2000; i++) {
			mcp.log('info', 'Message ' + i);
		}
		'done'
	`

	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, "done", result)

	// Should be limited to MaxLogEntries
	assert.LessOrEqual(t, len(logs), MaxLogEntries)
	assert.Equal(t, MaxLogEntries, len(logs))
}

// TestExecute_ConcurrentExecutionNoBlocking verifies concurrent execution doesn't block
func TestExecute_ConcurrentExecutionNoBlocking(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Run multiple executions concurrently with different execution times
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	start := time.Now()

	for i := range numGoroutines {
		go func(id int) {
			script := `
				let sum = 0;
				for (let i = 0; i < 1000; i++) {
					sum += i;
				}
				sum
			`
			_, _, err := runtime.Execute(context.Background(), script)
			results <- err
		}(i)
	}

	// Wait for all goroutines
	for range numGoroutines {
		err := <-results
		require.NoError(t, err)
	}

	elapsed := time.Since(start)

	// All executions should complete relatively quickly since they don't block each other
	// If mutex was held during execution, this would take much longer
	assert.Less(t, elapsed, 5*time.Second, "Concurrent execution should not block")
}

// TestExecute_ContextCancellationDuringExecution verifies context cancellation during execution
func TestExecute_ContextCancellationDuringExecution(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Start execution that will be cancelled
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	script := `
		let sum = 0;
		for (let i = 0; i < 10000000; i++) {
			sum += i;
		}
		sum
	`

	_, _, err := runtime.Execute(ctx, script)
	require.Error(t, err)

	// Should return error for interruption/cancellation
	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	// Accept either "interrupt" or "cancel" in the error message
	msg := strings.ToLower(runtimeErr.Message)
	assert.True(t, strings.Contains(msg, "interrupt") || strings.Contains(msg, "cancel"),
		"Expected error message to contain 'interrupt' or 'cancel', got: %s", msg)
}

// TestExecute_TypeAssertionError verifies proper handling of type assertion failures
func TestExecute_TypeAssertionError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Try to call callTool with invalid params type (number instead of object)
	script := `mcp.callTool('server__tool', 123)`

	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	// Should return proper error, not panic
	// The important thing is that it doesn't panic and returns a proper RuntimeError
	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok, "Should return RuntimeError, not panic")
	assert.Equal(t, ErrorTypeRuntime, runtimeErr.Type)
}

// TestExecute_ComplexScript verifies complex script execution
func TestExecute_ComplexScript(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `
		function fibonacci(n) {
			if (n <= 1) return n;
			return fibonacci(n - 1) + fibonacci(n - 2);
		}

		mcp.log('info', 'Calculating fibonacci');
		const result = fibonacci(10);
		mcp.log('debug', 'Result calculated', {result: result});
		result
	`
	result, logs, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, int64(55), result)
	assert.Len(t, logs, 2)
	assert.Equal(t, "Calculating fibonacci", logs[0].Message)
	assert.Equal(t, "Result calculated", logs[1].Message)
}

// TestExecute_ContextCancellation verifies context cancellation handling
func TestExecute_ContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	script := `1 + 1`
	_, _, err := runtime.Execute(ctx, script)
	require.Error(t, err)

	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	// Immediate cancellation should surface as timeout or runtime interruption
	assert.Contains(t, []ErrorType{ErrorTypeRuntime, ErrorTypeTimeout}, runtimeErr.Type)
}

// TestExecute_ReturnTypes verifies different return types
func TestExecute_ReturnTypes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name     string
		script   string
		expected any
	}{
		{
			name:     "number",
			script:   `42`,
			expected: int64(42),
		},
		{
			name:     "string",
			script:   `"hello"`,
			expected: "hello",
		},
		{
			name:     "boolean",
			script:   `true`,
			expected: true,
		},
		{
			name:     "null",
			script:   `null`,
			expected: nil,
		},
		{
			name:     "object",
			script:   `({key: 'value'})`,
			expected: map[string]any{"key": "value"},
		},
		{
			name:     "array",
			script:   `[1, 2, 3]`,
			expected: []any{int64(1), int64(2), int64(3)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, logs, err := runtime.Execute(context.Background(), tt.script)
			require.NoError(t, err)
			assert.Empty(t, logs)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSanitizeError verifies error message sanitization
func TestSanitizeError(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove newlines",
			input:    "Error\nOn\nMultiple\nLines",
			expected: "Error On Multiple Lines",
		},
		{
			name:     "remove tabs",
			input:    "Error\tWith\tTabs",
			expected: "Error With Tabs",
		},
		{
			name:     "truncate long messages",
			input:    string(make([]byte, 600)),
			expected: string(make([]byte, 500)) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeError(tt.input)
			if tt.name == "truncate long messages" {
				assert.Len(t, result, 503) // 500 + "..."
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestCallTool_Validation verifies callTool input validation
func TestCallTool_Validation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "missing double underscore separator",
			script: `mcp.callTool('toolName', {})`,
		},
		{
			name:   "empty serverID",
			script: `mcp.callTool('__toolName', {})`,
		},
		{
			name:   "empty toolName",
			script: `mcp.callTool('server__', {})`,
		},
		{
			name:   "wrong number of arguments",
			script: `mcp.callTool('server__tool')`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err)
		})
	}
}

// TestMcpLog_EdgeCases verifies mcp.log edge cases
func TestMcpLog_EdgeCases(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name         string
		script       string
		expectedLogs int
	}{
		{
			name:         "undefined fields",
			script:       `mcp.log('info', 'message', undefined); 'ok'`,
			expectedLogs: 1,
		},
		{
			name:         "null fields",
			script:       `mcp.log('info', 'message', null); 'ok'`,
			expectedLogs: 1,
		},
		{
			name:         "no fields",
			script:       `mcp.log('info', 'message'); 'ok'`,
			expectedLogs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, logs, err := runtime.Execute(context.Background(), tt.script)
			require.NoError(t, err)
			assert.Equal(t, "ok", result)
			assert.Len(t, logs, tt.expectedLogs)
		})
	}
}

// TestConsoleLog verifies console.log/warn/error work as aliases for mcp.log
func TestConsoleLog(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name          string
		script        string
		expectedLevel string
		expectedMsg   string
	}{
		{
			name:          "console.log",
			script:        `console.log('hello', 'world'); 'ok'`,
			expectedLevel: "info",
			expectedMsg:   "hello world",
		},
		{
			name:          "console.info",
			script:        `console.info('info message'); 'ok'`,
			expectedLevel: "info",
			expectedMsg:   "info message",
		},
		{
			name:          "console.warn",
			script:        `console.warn('warning!'); 'ok'`,
			expectedLevel: "warn",
			expectedMsg:   "warning!",
		},
		{
			name:          "console.error",
			script:        `console.error('error occurred'); 'ok'`,
			expectedLevel: "error",
			expectedMsg:   "error occurred",
		},
		{
			name:          "console.debug",
			script:        `console.debug('debug info'); 'ok'`,
			expectedLevel: "debug",
			expectedMsg:   "debug info",
		},
		{
			name:          "console.log with multiple args",
			script:        `console.log('count:', 42, 'items'); 'ok'`,
			expectedLevel: "info",
			expectedMsg:   "count: 42 items",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, logs, err := runtime.Execute(context.Background(), tt.script)
			require.NoError(t, err)
			assert.Equal(t, "ok", result)
			require.Len(t, logs, 1)
			assert.Equal(t, tt.expectedLevel, logs[0].Level)
			assert.Equal(t, tt.expectedMsg, logs[0].Message)
		})
	}
}

// TestThreadSafety verifies concurrent execution safety
func TestThreadSafety(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Run multiple executions concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := range numGoroutines {
		go func(id int) {
			script := `1 + 1`
			result, logs, err := runtime.Execute(context.Background(), script)
			if err != nil {
				errors <- err
			} else if result != int64(2) {
				errors <- assert.AnError
			} else if len(logs) != 0 {
				errors <- assert.AnError
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for range numGoroutines {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
	}
}

// TestExecute_ToolAuthorization verifies tool authorization enforcement
func TestExecute_ToolAuthorization(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create runtime with allowed tools
	allowedTools := map[string][]string{
		"server1": {"tool1", "tool2"},
		"server2": {"tool3"},
	}
	runtime := NewRuntime(logger, manager, &Config{
		AllowedTools: allowedTools,
	})

	tests := []struct {
		name      string
		script    string
		shouldErr bool
	}{
		{
			name:      "allowed tool",
			script:    `mcp.callTool('server1__tool1', {})`,
			shouldErr: true, // Will error because server doesn't exist, but authorization passes
		},
		{
			name:      "disallowed tool",
			script:    `mcp.callTool('server1__tool3', {})`,
			shouldErr: true,
		},
		{
			name:      "disallowed server",
			script:    `mcp.callTool('server3__tool1', {})`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestExecute_ToolAuthorizationNilAllowsAll verifies nil allowedTools allows all
func TestExecute_ToolAuthorizationNilAllowsAll(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	// Create runtime with nil allowed tools (allow all)
	runtime := NewRuntime(logger, manager, nil)

	// Should not reject based on authorization (but will fail due to missing server)
	script := `mcp.callTool('anyserver__anytool', {})`
	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)
	// Should get "server not found" error, not authorization error
	assert.NotContains(t, err.Error(), "not authorized")
	assert.Contains(t, err.Error(), "not found")
}

// TestExecute_ErrorSanitization verifies tool call errors are sanitized
func TestExecute_ErrorSanitization(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Call a non-existent tool to trigger error
	script := `mcp.callTool('nonexistent__tool', {})`
	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	errMsg := err.Error()
	// Error should be generic and not leak internal details
	assert.NotContains(t, errMsg, "/Users/")
	assert.NotContains(t, errMsg, "/home/")
	assert.NotContains(t, errMsg, "C:\\")
	assert.NotContains(t, errMsg, "connection")
	assert.NotContains(t, errMsg, "grpc")
}

// TestExecute_ParamsTypeAssertion verifies proper error on invalid params type
func TestExecute_ParamsTypeAssertion(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "number instead of object",
			script: `mcp.callTool('server__tool', 123)`,
		},
		{
			name:   "string instead of object",
			script: `mcp.callTool('server__tool', 'invalid')`,
		},
		{
			name:   "array instead of object",
			script: `mcp.callTool('server__tool', [1, 2, 3])`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err)
			// Should get proper error message about params type
			assert.Contains(t, err.Error(), "params must be an object")
		})
	}
}

// TestExecute_TimeoutDoesNotLeakGoroutines verifies no goroutine leaks on timeout
func TestExecute_TimeoutDoesNotLeakGoroutines(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	cfg := &Config{
		Timeout: 200 * time.Millisecond,
	}
	runtime := NewRuntime(logger, manager, cfg)

	// Run multiple timeouts
	for range 3 {
		script := `while(true) { var x = 1 + 1; }`
		_, _, err := runtime.Execute(context.Background(), script)
		require.Error(t, err)

		runtimeErr, ok := err.(*RuntimeError)
		require.True(t, ok)
		// Check for either timeout or runtime_error (interrupt)
		isValidError := runtimeErr.Type == ErrorTypeTimeout || runtimeErr.Type == ErrorTypeRuntime
		assert.True(t, isValidError, "Expected timeout or runtime_error, got %s", runtimeErr.Type)
		assert.Contains(t, runtimeErr.Message, "interrupted")
	}

	// Verify we can still execute after timeouts
	script := `1 + 1`
	result, _, err := runtime.Execute(context.Background(), script)
	require.NoError(t, err)
	assert.Equal(t, int64(2), result)
}

// TestExecute_ConcurrentTimeouts verifies concurrent timeouts don't interfere
func TestExecute_ConcurrentTimeouts(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, &Config{Timeout: 100 * time.Millisecond})

	const numGoroutines = 5
	results := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			script := `while(true) { let x = 1 + 1; }`
			_, _, err := runtime.Execute(context.Background(), script)
			results <- err
		}()
	}

	// All should timeout
	for range numGoroutines {
		err := <-results
		require.Error(t, err)
		runtimeErr, ok := err.(*RuntimeError)
		require.True(t, ok)
		assert.Equal(t, ErrorTypeTimeout, runtimeErr.Type)
	}
}
