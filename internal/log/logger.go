// Package log provides structured logging for gline using zerolog.
// It supports multiple log levels, file output, and console output with colors.
package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// Logger is the global logger instance
var Logger zerolog.Logger

// Level represents log levels
type Level int8

const (
	// DebugLevel logs everything
	DebugLevel Level = -4
	// InfoLevel logs info, warnings, and errors
	InfoLevel Level = 0
	// WarnLevel logs warnings and errors
	WarnLevel Level = 4
	// ErrorLevel logs only errors
	ErrorLevel Level = 8
	// FatalLevel logs fatal errors and exits
	FatalLevel Level = 12
)

// Config holds logger configuration
type Config struct {
	// Level is the minimum log level to output
	Level string

	// File is the path to the log file (empty for no file output)
	File string

	// Console enables console output
	Console bool

	// Color enables colored output in console
	Color bool
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Level:   "info",
		File:    "",
		Console: true,
		Color:   true,
	}
}

// Init initializes the global logger with the given configuration
func Init(config Config) error {
	// Set up zerolog
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	// Parse log level
	level, err := parseLevel(config.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}

	// Create writers
	writers := []io.Writer{}

	// Console writer
	if config.Console {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
			NoColor:    !config.Color,
		}
		writers = append(writers, consoleWriter)
	}

	// File writer
	if config.File != "" {
		// Create log directory if needed
		logDir := filepath.Dir(config.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		writers = append(writers, file)
	}

	// Create multi-writer
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else if len(writers) > 1 {
		output = io.MultiWriter(writers...)
	} else {
		output = io.Discard
	}

	// Create logger
	Logger = zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()

	return nil
}

// InitWithDefaults initializes the logger with default configuration
func InitWithDefaults() error {
	return Init(DefaultConfig())
}

// parseLevel parses a log level string
func parseLevel(level string) (zerolog.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warn", "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	case "fatal":
		return zerolog.FatalLevel, nil
	case "panic":
		return zerolog.PanicLevel, nil
	case "trace":
		return zerolog.TraceLevel, nil
	default:
		return zerolog.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

// Debug logs a debug message
func Debug(msg string) {
	Logger.Debug().Msg(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, v ...interface{}) {
	Logger.Debug().Msgf(format, v...)
}

// Info logs an info message
func Info(msg string) {
	Logger.Info().Msg(msg)
}

// Infof logs a formatted info message
func Infof(format string, v ...interface{}) {
	Logger.Info().Msgf(format, v...)
}

// Warn logs a warning message
func Warn(msg string) {
	Logger.Warn().Msg(msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, v ...interface{}) {
	Logger.Warn().Msgf(format, v...)
}

// Error logs an error message
func Error(msg string) {
	Logger.Error().Msg(msg)
}

// Errorf logs a formatted error message
func Errorf(format string, v ...interface{}) {
	Logger.Error().Msgf(format, v...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	Logger.Fatal().Msg(msg)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, v ...interface{}) {
	Logger.Fatal().Msgf(format, v...)
}

// With creates a child logger with additional fields
func With(key string, value interface{}) zerolog.Logger {
	return Logger.With().Interface(key, value).Logger()
}

// WithError creates a child logger with an error
func WithError(err error) *zerolog.Event {
	return Logger.Error().Err(err)
}

// WithField creates a child logger with a field
func WithField(key string, value interface{}) *zerolog.Event {
	return Logger.Info().Interface(key, value)
}

// SetLevel changes the log level dynamically
func SetLevel(level string) error {
	l, err := parseLevel(level)
	if err != nil {
		return err
	}
	Logger = Logger.Level(l)
	return nil
}

// GetLevel returns the current log level
func GetLevel() string {
	return Logger.GetLevel().String()
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	return Logger.GetLevel() <= zerolog.DebugLevel
}
