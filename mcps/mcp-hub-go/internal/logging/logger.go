package logging

import (
	"go.uber.org/zap"
)

// Logger is the global logger instance
var Logger *zap.Logger

// InitLogger initializes the global logger
func InitLogger(development bool) error {
	var err error
	if development {
		Logger, err = zap.NewDevelopment()
	} else {
		Logger, err = zap.NewProduction()
	}
	return err
}

// Sync flushes any buffered log entries
func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
