package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitLogger_StdoutOnly(t *testing.T) {
	// Test logger initialization with stdout only (no file logging)
	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: "", // No file logging
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.FileLoggingEnabled)
	assert.Nil(t, result.FileLoggingError)
	assert.NotNil(t, Logger)

	// Verify logger can log without errors
	Logger.Info("test message", zap.String("key", "value"))
	err = Sync()
	assert.NoError(t, err)
}

func TestInitLogger_WithFileLogging(t *testing.T) {
	// Create temp directory for log file
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.FileLoggingEnabled)
	assert.Nil(t, result.FileLoggingError)
	assert.NotNil(t, Logger)

	// Write a log message
	Logger.Info("test message", zap.String("key", "value"))
	err = Sync()
	assert.NoError(t, err)

	// Verify log file was created and contains the message
	assert.FileExists(t, logFile)
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test message")
	assert.Contains(t, string(content), "key")
	assert.Contains(t, string(content), "value")

	// Verify JSON format
	var logEntry map[string]interface{}
	err = json.Unmarshal(content, &logEntry)
	assert.NoError(t, err, "log output should be valid JSON")
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "test message", logEntry["msg"])

	// Verify file permissions are 0600 (owner-only read/write)
	fileInfo, err := os.Stat(logFile)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fileInfo.Mode().Perm())
}

func TestInitLogger_UnwritableFile_GracefulFallback(t *testing.T) {
	// Try to create a log file in a non-existent directory
	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: "/nonexistent/directory/test.log",
	}

	// Should not return an error - graceful fallback to stdout only
	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.FileLoggingEnabled)
	assert.NotNil(t, result.FileLoggingError)
	assert.NotNil(t, Logger)

	// Logger should still work (stdout only)
	Logger.Info("test message after fallback")
	err = Sync()
	assert.NoError(t, err)
}

func TestInitLogger_LogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    zapcore.Level
		logDebug bool
		logInfo  bool
		logWarn  bool
		logError bool
	}{
		{
			name:     "debug level - logs everything",
			level:    zapcore.DebugLevel,
			logDebug: true,
			logInfo:  true,
			logWarn:  true,
			logError: true,
		},
		{
			name:     "info level - logs info and above",
			level:    zapcore.InfoLevel,
			logDebug: false,
			logInfo:  true,
			logWarn:  true,
			logError: true,
		},
		{
			name:     "warn level - logs warn and above",
			level:    zapcore.WarnLevel,
			logDebug: false,
			logInfo:  false,
			logWarn:  true,
			logError: true,
		},
		{
			name:     "error level - logs only errors",
			level:    zapcore.ErrorLevel,
			logDebug: false,
			logInfo:  false,
			logWarn:  false,
			logError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			logFile := filepath.Join(tmpDir, "test.log")

			cfg := Config{
				LogLevel:    tt.level,
				LogFilePath: logFile,
			}

			result, err := InitLogger(cfg)
			assert.NoError(t, err)
			assert.True(t, result.FileLoggingEnabled)

			// Log at all levels
			Logger.Debug("debug message")
			Logger.Info("info message")
			Logger.Warn("warn message")
			Logger.Error("error message")
			err = Sync()
			assert.NoError(t, err)

			// Read log file
			content, err := os.ReadFile(logFile)
			assert.NoError(t, err)

			// Verify expected messages are present
			if tt.logDebug {
				assert.Contains(t, string(content), "debug message")
			} else {
				assert.NotContains(t, string(content), "debug message")
			}

			if tt.logInfo {
				assert.Contains(t, string(content), "info message")
			} else {
				assert.NotContains(t, string(content), "info message")
			}

			if tt.logWarn {
				assert.Contains(t, string(content), "warn message")
			} else {
				assert.NotContains(t, string(content), "warn message")
			}

			if tt.logError {
				assert.Contains(t, string(content), "error message")
			}
		})
	}
}

func TestInitLogger_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)

	// Log a structured message
	Logger.Info("structured log",
		zap.String("string_field", "value"),
		zap.Int("int_field", 42),
		zap.Bool("bool_field", true),
	)
	err = Sync()
	assert.NoError(t, err)

	// Read log file
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)

	// Parse JSON
	var logEntry map[string]interface{}
	err = json.Unmarshal(content, &logEntry)
	assert.NoError(t, err, "log output should be valid JSON")

	// Verify JSON structure
	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, "structured log", logEntry["msg"])
	assert.Equal(t, "value", logEntry["string_field"])
	assert.Equal(t, float64(42), logEntry["int_field"]) // JSON numbers are float64
	assert.Equal(t, true, logEntry["bool_field"])
	assert.NotEmpty(t, logEntry["timestamp"])
	assert.NotEmpty(t, logEntry["caller"])
}

func TestInitLogger_AppendToExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	// Create initial log entry
	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)
	Logger.Info("first message")
	err = Sync()
	assert.NoError(t, err)

	// Reinitialize logger (simulating restart)
	result, err = InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)
	Logger.Info("second message")
	err = Sync()
	assert.NoError(t, err)

	// Verify both messages are in the file
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "first message")
	assert.Contains(t, string(content), "second message")

	// Verify we have two JSON lines
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	assert.Equal(t, 2, len(lines))
}

func TestSync(t *testing.T) {
	// Test Sync with nil logger
	Logger = nil
	loggerState.logger = nil
	err := Sync()
	assert.NoError(t, err) // Should not panic and return nil error

	// Test Sync with initialized logger
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, initErr := InitLogger(cfg)
	assert.NoError(t, initErr)
	assert.True(t, result.FileLoggingEnabled)

	Logger.Info("test message")
	err = Sync() // Should flush to file
	assert.NoError(t, err)

	// Verify message was written
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test message")
}

func TestWithRequestID(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)

	// Create logger with request ID
	requestID := "test-request-123"
	reqLogger := WithRequestID(requestID)
	assert.NotNil(t, reqLogger)

	// Log with request ID
	reqLogger.Info("handling request")
	reqLogger.Sync()

	// Verify request_id is in the log
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(content, &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, requestID, logEntry["request_id"])
}

func TestWithRequestID_NilLogger(t *testing.T) {
	// Test WithRequestID when Logger is nil
	Logger = nil
	loggerState.logger = nil
	reqLogger := WithRequestID("test-id")
	assert.NotNil(t, reqLogger) // Should return zap.NewNop() instead of nil
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  zapcore.Level
		expectErr bool
	}{
		{
			name:      "debug level",
			input:     "debug",
			expected:  zapcore.DebugLevel,
			expectErr: false,
		},
		{
			name:      "info level",
			input:     "info",
			expected:  zapcore.InfoLevel,
			expectErr: false,
		},
		{
			name:      "warn level",
			input:     "warn",
			expected:  zapcore.WarnLevel,
			expectErr: false,
		},
		{
			name:      "error level",
			input:     "error",
			expected:  zapcore.ErrorLevel,
			expectErr: false,
		},
		{
			name:      "uppercase",
			input:     "INFO",
			expected:  zapcore.InfoLevel,
			expectErr: false,
		},
		{
			name:      "invalid level",
			input:     "invalid",
			expected:  zapcore.InfoLevel,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level, err := ParseLevel(tt.input)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, level)
			}
		})
	}
}

func TestInitLogger_CallerInformation(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)

	Logger.Info("test message with caller")
	err = Sync()
	assert.NoError(t, err)

	// Verify caller information is present
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(content, &logEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, logEntry["caller"])
	assert.Contains(t, logEntry["caller"], "logger_test.go")
}

func TestInitLogger_Stacktrace(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.DebugLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)

	// Error level should include stacktrace
	Logger.Error("error with stacktrace")
	err = Sync()
	assert.NoError(t, err)

	// Verify stacktrace is present for error
	content, err := os.ReadFile(logFile)
	assert.NoError(t, err)

	var logEntry map[string]interface{}
	err = json.Unmarshal(content, &logEntry)
	assert.NoError(t, err)
	assert.NotEmpty(t, logEntry["stacktrace"])
}

func TestInitLogger_MultipleInits(t *testing.T) {
	// Test that multiple InitLogger calls don't cause issues
	tmpDir := t.TempDir()

	cfg1 := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: filepath.Join(tmpDir, "log1.log"),
	}

	result, err := InitLogger(cfg1)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)
	Logger.Info("first init")
	err = Sync()
	assert.NoError(t, err)

	cfg2 := Config{
		LogLevel:    zapcore.DebugLevel,
		LogFilePath: filepath.Join(tmpDir, "log2.log"),
	}

	result, err = InitLogger(cfg2)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)
	Logger.Debug("second init")
	err = Sync()
	assert.NoError(t, err)

	// Verify second log file has the debug message
	content, err := os.ReadFile(cfg2.LogFilePath)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "second init")
}

func TestInitLogger_PathSecurity(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name                string
		logFilePath         string
		expectFileLogging   bool
	}{
		{
			name:              "path with .. is cleaned and converted to absolute",
			logFilePath:       filepath.Join(tmpDir, "..", filepath.Base(tmpDir), "test.log"),
			expectFileLogging: true,
		},
		{
			name:              "relative path should be converted to absolute",
			logFilePath:       "test.log",
			expectFileLogging: true,
		},
		{
			name:              "absolute path should work",
			logFilePath:       filepath.Join(tmpDir, "test.log"),
			expectFileLogging: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				LogLevel:    zapcore.InfoLevel,
				LogFilePath: tt.logFilePath,
			}

			result, err := InitLogger(cfg)
			assert.NoError(t, err) // InitLogger should never return error

			assert.Equal(t, tt.expectFileLogging, result.FileLoggingEnabled)
		})
	}
}

func TestGetLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		LogLevel:    zapcore.InfoLevel,
		LogFilePath: logFile,
	}

	result, err := InitLogger(cfg)
	assert.NoError(t, err)
	assert.True(t, result.FileLoggingEnabled)

	// GetLogger should return the same logger instance
	logger := GetLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, Logger, logger)
}
