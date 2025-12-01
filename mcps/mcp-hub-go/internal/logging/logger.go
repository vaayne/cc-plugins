package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// loggerState holds the global logger state with thread-safe access
var loggerState = struct {
	mu     sync.Mutex
	logger *zap.Logger
	file   *os.File
}{
	logger: nil,
	file:   nil,
}

// Logger is the global logger instance (deprecated: use GetLogger() for thread-safe access)
var Logger *zap.Logger

// Config holds logging configuration
type Config struct {
	// LogLevel sets the minimum log level (debug, info, warn, error)
	LogLevel zapcore.Level
	// LogFilePath is the path to the log file (empty to disable file logging)
	LogFilePath string
}

// InitResult contains the result of logger initialization
type InitResult struct {
	// FileLoggingEnabled indicates whether file logging was successfully enabled
	FileLoggingEnabled bool
	// FileLoggingError contains the error if file logging failed (nil if succeeded or not attempted)
	FileLoggingError error
}

// InitLogger initializes the global logger with JSON structured logging
// It always logs to stdout and optionally logs to a file if LogFilePath is set
// Returns InitResult indicating whether file logging was enabled and any errors encountered
func InitLogger(cfg Config) (*InitResult, error) {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	result := &InitResult{
		FileLoggingEnabled: false,
		FileLoggingError:   nil,
	}

	// Close previous file handle if exists to prevent file descriptor leak
	if loggerState.file != nil {
		if err := loggerState.file.Close(); err != nil {
			// Log warning but continue - we're reinitializing anyway
			if loggerState.logger != nil {
				loggerState.logger.Warn("Failed to close previous log file during reinitialization",
					zap.Error(err),
				)
			}
		}
		loggerState.file = nil
	}

	// Create JSON encoder config for production
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	// Create stdout core (always enabled)
	stdoutCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		cfg.LogLevel,
	)

	// Try to create file core if log file path is specified
	var cores []zapcore.Core
	cores = append(cores, stdoutCore)

	if cfg.LogFilePath != "" {
		fileCore, file, err := createFileCore(cfg.LogFilePath, encoderConfig, cfg.LogLevel)
		if err != nil {
			// Log warning to stdout and continue without file logging
			tempLogger := zap.New(stdoutCore)
			tempLogger.Warn("Failed to initialize file logging, continuing with stdout only",
				zap.String("log_file", cfg.LogFilePath),
				zap.Error(err),
			)
			result.FileLoggingError = err
		} else {
			cores = append(cores, fileCore)
			loggerState.file = file
			result.FileLoggingEnabled = true
		}
	}

	// Combine cores using a tee
	core := zapcore.NewTee(cores...)

	// Create logger with caller information
	loggerState.logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	Logger = loggerState.logger

	return result, nil
}

// validateLogFilePath validates and sanitizes the log file path
func validateLogFilePath(path string) (string, error) {
	// Clean the path to normalize it (removes . and .. where possible)
	cleanPath := filepath.Clean(path)

	// Convert to absolute path for security and clarity
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert to absolute path: %w", err)
	}

	return absPath, nil
}

// createFileCore creates a file-based logging core
func createFileCore(logFilePath string, encoderConfig zapcore.EncoderConfig, level zapcore.Level) (zapcore.Core, *os.File, error) {
	// Validate and sanitize the file path
	cleanPath, err := validateLogFilePath(logFilePath)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid log file path: %w", err)
	}

	// Try to open/create the log file with restricted permissions (owner-only read/write)
	file, err := os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create file core
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(file),
		level,
	)

	return fileCore, file, nil
}

// Sync flushes any buffered log entries
func Sync() error {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	if loggerState.logger != nil {
		// Sync can fail on stdout/stderr with "bad file descriptor" in tests
		// This is a known issue with zap - ignore these specific errors
		if err := loggerState.logger.Sync(); err != nil {
			// Ignore stdout/stderr sync errors
			if strings.Contains(err.Error(), "/dev/stdout") || 
				strings.Contains(err.Error(), "/dev/stderr") ||
				strings.Contains(err.Error(), "invalid argument") {
				return nil
			}
			return err
		}
	}
	return nil
}

// GetLogger returns the global logger instance in a thread-safe manner
func GetLogger() *zap.Logger {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()
	return loggerState.logger
}

// WithRequestID returns a logger with the request ID field added
// This is useful for tracing requests across log entries
func WithRequestID(requestID string) *zap.Logger {
	loggerState.mu.Lock()
	defer loggerState.mu.Unlock()

	if loggerState.logger == nil {
		return zap.NewNop()
	}
	return loggerState.logger.With(zap.String("request_id", requestID))
}

// ParseLevel converts a string log level to zapcore.Level
func ParseLevel(level string) (zapcore.Level, error) {
	var l zapcore.Level
	err := l.UnmarshalText([]byte(level))
	return l, err
}
