package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

// LoggerInterface 定义日志接口。LoggerInterface defines the logging interface.
type LoggerInterface interface {
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	// 更多级别。More levels.
}

// Logger 是全局日志实例。Logger is the global logging instance.
var Logger *logrus.Logger

// InitLogger 初始化日志记录器。InitLogger initializes the logger.
func InitLogger() {
	Logger = logrus.New()
	Logger.SetOutput(os.Stdout)
	Logger.SetLevel(logrus.InfoLevel)
	Logger.SetFormatter(&logrus.JSONFormatter{})
}

//Personal.AI order the ending
