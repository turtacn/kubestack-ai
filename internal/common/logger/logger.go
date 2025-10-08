// Copyright © 2024 KubeStack-AI Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package logger provides a flexible and structured logging framework for KubeStack-AI.
// It is built on top of logrus and supports various formats, outputs, and log rotation.
package logger

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger defines the standard logging interface used across the application.
// This abstraction allows the underlying logging implementation (e.g., logrus)
// to be swapped out without changing application code.
type Logger interface {
	// WithField adds a single structured field to the logger's context.
	WithField(key string, value interface{}) Logger
	// WithFields adds multiple structured fields to the logger's context.
	WithFields(fields map[string]interface{}) Logger
	// Debug logs a message at the debug level.
	Debug(args ...interface{})
	// Info logs a message at the info level.
	Info(args ...interface{})
	// Warn logs a message at the warning level.
	Warn(args ...interface{})
	// Error logs a message at the error level.
	Error(args ...interface{})
	// Fatal logs a message at the fatal level and then calls os.Exit(1).
	Fatal(args ...interface{})
	// Debugf logs a formatted message at the debug level.
	Debugf(format string, args ...interface{})
	// Infof logs a formatted message at the info level.
	Infof(format string, args ...interface{})
	// Warnf logs a formatted message at the warning level.
	Warnf(format string, args ...interface{})
	// Errorf logs a formatted message at the error level.
	Errorf(format string, args ...interface{})
	// Fatalf logs a formatted message at the fatal level and then calls os.Exit(1).
	Fatalf(format string, args ...interface{})
}

// logrusLogger is a concrete implementation of the Logger interface using logrus.
type logrusLogger struct {
	entry *logrus.Entry
}

// Config holds all configuration for the logger, allowing for detailed control
// over log levels, formats, outputs, and rotation policies.
type Config struct {
	// Level is the minimum logging level to output (e.g., "debug", "info", "warn").
	Level string `mapstructure:"level"`
	// Format specifies the log format, either "json" for structured logs or "text" for human-readable logs.
	Format string `mapstructure:"format"`
	// Output defines where logs should be sent: "console", "file", or "both".
	Output string `mapstructure:"output"`
	// File is the path to the log file, used when Output is "file" or "both".
	File string `mapstructure:"file"`
	// MaxSize is the maximum size in megabytes of a log file before it gets rotated.
	MaxSize int `mapstructure:"maxSize"`
	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int `mapstructure:"maxBackups"`
	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int `mapstructure:"maxAge"`
	// Compress determines whether to compress rotated log files using gzip.
	Compress bool `mapstructure:"compress"`
}

var globalLogger Logger = &logrusLogger{entry: logrus.NewEntry(logrus.New())}

// InitGlobalLogger initializes the global logger instance with the given configuration.
// It configures the logging level, format, output (console, file, or both), and
// sets up log rotation using the lumberjack library. This function should be
// called once at application startup and is not thread-safe.
//
// Parameters:
//   cfg (*Config): A pointer to the logger configuration struct.
func InitGlobalLogger(cfg *Config) {
	l := logrus.New()

	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	l.SetLevel(level)

	if cfg.Format == "json" {
		l.SetFormatter(&logrus.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.000"})
	} else {
		l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, TimestampFormat: "2006-01-02 15:04:05.000"})
	}

	var writers []io.Writer
	if cfg.Output == "file" || cfg.Output == "both" {
		writers = append(writers, &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
	}
	if cfg.Output == "console" || cfg.Output == "both" || len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}
	l.SetOutput(io.MultiWriter(writers...))

	// TODO: Add syslog hook if needed, which might require another dependency.
	// Example:
	// if cfg.Syslog.Enable {
	// 	hook, err := logrus_syslog.NewSyslogHook(cfg.Syslog.Network, cfg.Syslog.Address, syslog.LOG_INFO, "")
	// 	if err == nil {
	// 		l.Hooks.Add(hook)
	// 	}
	// }

	globalLogger = &logrusLogger{entry: logrus.NewEntry(l)}
}

// GetLogger returns the configured global logger instance. This function provides
// a convenient way to access the logger from anywhere in the application after it
// has been initialized.
//
// Returns:
//   Logger: The singleton logger instance.
func GetLogger() Logger {
	return globalLogger
}

// NewLogger returns a new logger with a "module" field already added to its
// context. This is a factory function for creating contextual loggers that
// automatically tag log entries with the name of the component or module.
//
// Parameters:
//   module (string): The name of the module to be added as a field.
//
// Returns:
//   Logger: A new logger instance with the "module" field.
func NewLogger(module string) Logger {
	return GetLogger().WithField("module", module)
}

// WithField returns a new logger entry with the specified field added to its context.
// This is used to add structured data to log messages.
func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{entry: l.entry.WithField(key, value)}
}

// WithFields returns a new logger entry with the specified fields added to its context.
// This is used to add multiple structured data fields to log messages.
func (l *logrusLogger) WithFields(fields map[string]interface{}) Logger {
	return &logrusLogger{entry: l.entry.WithFields(fields)}
}

func (l *logrusLogger) Debug(args ...interface{}) { l.entry.Debug(args...) }
func (l *logrusLogger) Info(args ...interface{})  { l.entry.Info(args...) }
func (l *logrusLogger) Warn(args ...interface{})  { l.entry.Warn(args...) }
func (l *logrusLogger) Error(args ...interface{}) { l.entry.Error(args...) }
func (l *logrusLogger) Fatal(args ...interface{}) { l.entry.Fatal(args...) }

func (l *logrusLogger) Debugf(format string, args ...interface{}) { l.entry.Debugf(format, args...) }
func (l *logrusLogger) Infof(format string, args ...interface{})  { l.entry.Infof(format, args...) }
func (l *logrusLogger) Warnf(format string, args ...interface{})  { l.entry.Warnf(format, args...) }
func (l *logrusLogger) Errorf(format string, args ...interface{}) { l.entry.Errorf(format, args...) }
func (l *logrusLogger) Fatalf(format string, args ...interface{}) { l.entry.Fatalf(format, args...) }

//Personal.AI order the ending
