package logging

import (
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// LoggerInterface 定义日志接口。LoggerInterface defines the logging interface.
type LoggerInterface interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// Logger 是全局日志实例。Logger is the global logging instance.
var Logger *logrus.Logger

// InitLogger 初始化日志记录器。InitLogger initializes the logger.
func InitLogger() {
	Logger = logrus.New()

	// 设置日志输出：同时输出到控制台和文件。Set log output: both console and file.
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		Logger.Warn("Failed to create log directory", err)
		Logger.SetOutput(os.Stdout)
	} else {
		logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			Logger.Warn("Failed to open log file", err)
			Logger.SetOutput(os.Stdout)
		} else {
			Logger.SetOutput(os.Stdout)
		}
	}

	// 设置日志级别。Set log level.
	Logger.SetLevel(logrus.InfoLevel)

	// 设置日志格式为JSON。Set log format to JSON.
	Logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyTime:  "time",
			logrus.FieldKeyMsg:   "message",
		},
	})
}

//Personal.AI order the ending
