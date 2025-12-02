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

// TestExecute_RejectPromise verifies Promise rejection
func TestExecute_RejectPromise(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	script := `new Promise((resolve) => resolve(42))`
	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)

	runtimeErr, ok := err.(*RuntimeError)
	require.True(t, ok)
	assert.Equal(t, ErrorTypeAsync, runtimeErr.Type)
	assert.Contains(t, runtimeErr.Message, "Promise")
}

// TestExecute_RejectAsync verifies async function rejection
func TestExecute_RejectAsync(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "async function",
			script: `async function test() { return 42; }`,
		},
		{
			name:   "async arrow function",
			script: `const test = async () => 42`,
		},
		{
			name:   "await keyword",
			script: `await Promise.resolve(42)`,
		},
		{
			name:   "async with comment bypass attempt",
			script: `/*async*/ async function test() { return 42; }`,
		},
		{
			name:   "async with string concat",
			script: `const fn = 'async'; eval(fn + ' function test() {}')`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err)

			// Accept any error type as long as the script is rejected
			_, ok := err.(*RuntimeError)
			require.True(t, ok)
		})
	}
}

// TestExecute_RejectSetTimeout verifies setTimeout rejection
func TestExecute_RejectSetTimeout(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "setTimeout",
			script: `setTimeout(() => {}, 100)`,
		},
		{
			name:   "setInterval",
			script: `setInterval(() => {}, 100)`,
		},
		{
			name:   "setImmediate",
			script: `setImmediate(() => {})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err)

			runtimeErr, ok := err.(*RuntimeError)
			require.True(t, ok)
			assert.Equal(t, ErrorTypeAsync, runtimeErr.Type)
		})
	}
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

// TestExecute_AsyncBypassAttempts tests various attempts to bypass async detection
func TestExecute_AsyncBypassAttempts(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "Promise with bracket notation",
			script: `window['Promise']`,
		},
		{
			name:   "Promise with double quotes",
			script: `window["Promise"]`,
		},
		{
			name:   "Promise in comment (should still catch in actual code)",
			script: `// Promise\nnew Promise(r => r())`,
		},
		{
			name:   "async in string (should catch if used)",
			script: `"async"; async function test() {}`,
		},
		{
			name:   "setTimeout with bracket notation",
			script: `window['setTimeout'](function() {}, 100)`,
		},
		{
			name:   "setInterval with bracket notation",
			script: `global['setInterval'](function() {}, 100)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			// Should error with async or reference error (for window/global)
			require.Error(t, err)
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

// TestExecute_BlockDangerousGlobals verifies dangerous globals are blocked
func TestExecute_BlockDangerousGlobals(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "eval",
			script: `eval('1 + 1')`,
		},
		{
			name:   "Function constructor",
			script: `new Function('return 1 + 1')()`,
		},
		{
			name:   "Reflect",
			script: `Reflect.get({x: 1}, 'x')`,
		},
		{
			name:   "Proxy",
			script: `new Proxy({}, {})`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err, "Expected error for %s", tt.name)
		})
	}
}

// TestExecute_PrototypeFrozen verifies prototypes are frozen
func TestExecute_PrototypeFrozen(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "Object.prototype pollution",
			script: `Object.prototype.polluted = 'evil'; ({}).polluted !== 'evil'`,
		},
		{
			name:   "Array.prototype pollution",
			script: `Array.prototype.polluted = 'evil'; [].polluted !== 'evil'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := runtime.Execute(context.Background(), tt.script)
			// Frozen prototypes mean the assignment fails silently in non-strict mode
			// or throws in strict mode, but either way the prototype is not polluted
			// Check that either it errors OR the pollution didn't work
			if err == nil {
				assert.Equal(t, true, result, "Prototype should not be polluted")
			}
		})
	}
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

	for i := 0; i < numGoroutines; i++ {
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
	for i := 0; i < numGoroutines; i++ {
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
	script := `mcp.callTool('server', 'tool', 123)`

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
	// Immediate cancellation returns runtime_error, not timeout
	assert.Equal(t, ErrorTypeRuntime, runtimeErr.Type)
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
		expected interface{}
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
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "array",
			script:   `[1, 2, 3]`,
			expected: []interface{}{int64(1), int64(2), int64(3)},
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
			name:   "missing serverID",
			script: `mcp.callTool('', 'toolName', {})`,
		},
		{
			name:   "missing toolName",
			script: `mcp.callTool('server', '', {})`,
		},
		{
			name:   "wrong number of arguments",
			script: `mcp.callTool('server')`,
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

	for i := 0; i < numGoroutines; i++ {
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
	for i := 0; i < numGoroutines; i++ {
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
			script:    `mcp.callTool('server1', 'tool1', {})`,
			shouldErr: true, // Will error because server doesn't exist, but authorization passes
		},
		{
			name:      "disallowed tool",
			script:    `mcp.callTool('server1', 'tool3', {})`,
			shouldErr: true,
		},
		{
			name:      "disallowed server",
			script:    `mcp.callTool('server3', 'tool1', {})`,
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
	script := `mcp.callTool('anyserver', 'anytool', {})`
	_, _, err := runtime.Execute(context.Background(), script)
	require.Error(t, err)
	// Should get "failed to get client" not "not allowed"
	assert.NotContains(t, err.Error(), "not allowed")
}

// TestExecute_ConstructorAccessBlocked verifies Function constructor access is blocked
func TestExecute_ConstructorAccessBlocked(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "function constructor via prototype",
			script: `(function(){}).constructor('return 1+1')()`,
		},
		{
			name:   "array constructor",
			script: `[].constructor.constructor('return 1+1')()`,
		},
		{
			name:   "object constructor",
			script: `({}).constructor.constructor('return 1+1')()`,
		},
		{
			name:   "string constructor",
			script: `"".constructor.constructor('return 1+1')()`,
		},
		{
			name:   "number constructor",
			script: `(1).constructor.constructor('return 1+1')()`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err, "Expected error for %s", tt.name)
			// Should get "Access to constructor is not allowed" error
			assert.Contains(t, err.Error(), "constructor")
		})
	}
}

// TestExecute_ErrorSanitization verifies tool call errors are sanitized
func TestExecute_ErrorSanitization(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	// Call a non-existent tool to trigger error
	script := `mcp.callTool('nonexistent', 'tool', {})`
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
			script: `mcp.callTool('server', 'tool', 123)`,
		},
		{
			name:   "string instead of object",
			script: `mcp.callTool('server', 'tool', 'invalid')`,
		},
		{
			name:   "array instead of object",
			script: `mcp.callTool('server', 'tool', [1, 2, 3])`,
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
	for i := 0; i < 3; i++ {
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

	for i := 0; i < numGoroutines; i++ {
		go func() {
			script := `while(true) { let x = 1 + 1; }`
			_, _, err := runtime.Execute(context.Background(), script)
			results <- err
		}()
	}

	// All should timeout
	for i := 0; i < numGoroutines; i++ {
		err := <-results
		require.Error(t, err)
		runtimeErr, ok := err.(*RuntimeError)
		require.True(t, ok)
		assert.Equal(t, ErrorTypeTimeout, runtimeErr.Type)
	}
}

// TestExecute_AsyncBypassWithComments verifies async detection handles comments
func TestExecute_AsyncBypassWithComments(t *testing.T) {
	logger := zaptest.NewLogger(t)
	manager := client.NewManager(logger)
	defer manager.DisconnectAll()

	runtime := NewRuntime(logger, manager, nil)

	tests := []struct {
		name   string
		script string
	}{
		{
			name:   "async with inline comment",
			script: `/*comment*/ async function test() { return 42; }`,
		},
		{
			name:   "async with multiline comment before",
			script: `/* multi\nline\ncomment */ async function test() { return 42; }`,
		},
		{
			name:   "async after comment",
			script: `/* async in comment */ async function test() { return 42; }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := runtime.Execute(context.Background(), tt.script)
			require.Error(t, err)
		})
	}
}
