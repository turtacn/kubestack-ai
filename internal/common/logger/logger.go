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
type Logger interface {
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Fatalf(format string, args ...interface{})
}

// logrusLogger is a concrete implementation of the Logger interface using logrus.
type logrusLogger struct {
	entry *logrus.Entry
}

// Config holds all configuration for the logger.
type Config struct {
	Level      string `mapstructure:"level"`      // Logging level: "debug", "info", "warn", "error", "fatal"
	Format     string `mapstructure:"format"`     // Log format: "json" or "text"
	Output     string `mapstructure:"output"`     // Output target: "console", "file", "both"
	File       string `mapstructure:"file"`       // Log file path
	MaxSize    int    `mapstructure:"maxSize"`    // Max size in MB of a log file before it gets rotated
	MaxBackups int    `mapstructure:"maxBackups"` // Max number of old log files to retain
	MaxAge     int    `mapstructure:"maxAge"`     // Max number of days to retain old log files
	Compress   bool   `mapstructure:"compress"`   // Whether to compress old log files
}

var globalLogger Logger = &logrusLogger{entry: logrus.NewEntry(logrus.New())}

// InitGlobalLogger initializes the global logger instance with the given configuration.
// It is not thread-safe and should be called once at application startup.
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

// GetLogger returns the configured global logger.
func GetLogger() Logger {
	return globalLogger
}

// NewLogger returns a new logger with a "module" field, useful for contextual logging.
func NewLogger(module string) Logger {
	return GetLogger().WithField("module", module)
}

func (l *logrusLogger) WithField(key string, value interface{}) Logger {
	return &logrusLogger{entry: l.entry.WithField(key, value)}
}

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
