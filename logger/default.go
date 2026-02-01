package logger

import (
	"go.uber.org/zap"
)

// DefaultLogger is the default logger instance
var DefaultLogger Logger

func init() {
	// Initialize with production config by default
	zapLogger, _ := zap.NewProduction()
	DefaultLogger = NewZapLogger(zapLogger)
}

// SetDefault sets the default logger instance
func SetDefault(l Logger) {
	DefaultLogger = l
}

// Global convenience functions using DefaultLogger

func Debug(msg string, args ...Field) {
	DefaultLogger.Debug(msg, args...)
}

func Info(msg string, args ...Field) {
	DefaultLogger.Info(msg, args...)
}

func Warn(msg string, args ...Field) {
	DefaultLogger.Warn(msg, args...)
}

func Error(msg string, args ...Field) {
	DefaultLogger.Error(msg, args...)
}

func Fatal(msg string, args ...Field) {
	DefaultLogger.Fatal(msg, args...)
}

func Panic(msg string, args ...Field) {
	DefaultLogger.Panic(msg, args...)
}

func With(args ...Field) Logger {
	return DefaultLogger.With(args...)
}

func Sync() error {
	return DefaultLogger.Sync()
}
