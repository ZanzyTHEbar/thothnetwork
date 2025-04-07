package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for logging
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Info(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
	Error(msg string, keysAndValues ...any)
	Fatal(msg string, keysAndValues ...any)
	With(keysAndValues ...any) Logger
}

// zapLogger implements the Logger interface using zap
type zapLogger struct {
	logger *zap.SugaredLogger
}

// Config holds configuration for the logger
type Config struct {
	Level  string
	Format string
	Output string
}

// NewLogger creates a new logger
func NewLogger(config Config) (Logger, error) {
	// Parse log level
	var level zapcore.Level
	switch strings.ToLower(config.Level) {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn", "warning":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	case "fatal":
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create encoder based on format
	var encoder zapcore.Encoder
	switch strings.ToLower(config.Format) {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create writer based on output
	var writer zapcore.WriteSyncer
	switch strings.ToLower(config.Output) {
	case "stdout":
		writer = zapcore.AddSync(os.Stdout)
	case "stderr":
		writer = zapcore.AddSync(os.Stderr)
	default:
		// Try to open file
		file, err := os.OpenFile(config.Output, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		writer = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewCore(
		encoder,
		writer,
		zap.NewAtomicLevelAt(level),
	)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return &zapLogger{logger: logger.Sugar()}, nil
}

// Debug logs a debug message
func (l *zapLogger) Debug(msg string, keysAndValues ...any) {
	l.logger.Debugw(msg, keysAndValues...)
}

// Info logs an info message
func (l *zapLogger) Info(msg string, keysAndValues ...any) {
	l.logger.Infow(msg, keysAndValues...)
}

// Warn logs a warning message
func (l *zapLogger) Warn(msg string, keysAndValues ...any) {
	l.logger.Warnw(msg, keysAndValues...)
}

// Error logs an error message
func (l *zapLogger) Error(msg string, keysAndValues ...any) {
	l.logger.Errorw(msg, keysAndValues...)
}

// Fatal logs a fatal message and exits
func (l *zapLogger) Fatal(msg string, keysAndValues ...any) {
	l.logger.Fatalw(msg, keysAndValues...)
}

// With returns a logger with the specified key-value pairs
func (l *zapLogger) With(keysAndValues ...any) Logger {
	return &zapLogger{logger: l.logger.With(keysAndValues...)}
}

// NewDefaultLogger creates a new logger with default configuration
func NewDefaultLogger() Logger {
	logger, _ := NewLogger(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	return logger
}
