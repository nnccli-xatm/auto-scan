// 日志系统
// 特性：结构化日志、分级、文件轮转、审计日志分离

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger 日志器
type Logger struct {
	*logrus.Logger
	auditLogger *logrus.Logger
}

// NewLogger 创建日志器
func NewLogger() *Logger {
	logger := logrus.New()

	// 设置格式
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// 设置输出
	logger.SetOutput(os.Stdout)

	// 设置级别
	logger.SetLevel(logrus.InfoLevel)

	return &Logger{
		Logger: logger,
	}
}

// NewLoggerWithConfig 根据配置创建日志器
func NewLoggerWithConfig(level, format, output, filePath string) (*Logger, error) {
	logger := logrus.New()

	// 设置级别
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(lvl)

	// 设置格式
	switch format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s", format)
	}

	// 设置输出
	switch output {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "file":
		if filePath == "" {
			return nil, fmt.Errorf("log file path cannot be empty")
		}
		// 确保目录存在
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.SetOutput(file)
	default:
		return nil, fmt.Errorf("invalid log output: %s", output)
	}

	return &Logger{Logger: logger}, nil
}

// InitAuditLogger 初始化审计日志
func (l *Logger) InitAuditLogger(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("audit log file path cannot be empty")
	}

	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// 创建审计日志文件
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	auditLogger := logrus.New()
	auditLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})
	auditLogger.SetOutput(file)
	auditLogger.SetLevel(logrus.InfoLevel)

	l.auditLogger = auditLogger
	return nil
}

// Audit 记录审计日志
func (l *Logger) Audit(eventType, userID, deviceID, taskID, message string, details map[string]interface{}) {
	if l.auditLogger == nil {
		// 如果审计日志未初始化，使用普通日志记录
		l.WithFields(logrus.Fields{
			"event_type": eventType,
			"user_id":    userID,
			"device_id":  deviceID,
			"task_id":    taskID,
			"details":    details,
		}).Info(message)
		return
	}

	entry := l.auditLogger.WithFields(logrus.Fields{
		"event_type": eventType,
		"user_id":    userID,
		"device_id":  deviceID,
		"task_id":    taskID,
		"details":    details,
	})

	entry.Info(message)
}

// LogDeviceEvent 记录设备事件
func (l *Logger) LogDeviceEvent(deviceID, event string, fields map[string]interface{}) {
	entry := l.WithFields(logrus.Fields{
		"component": "device",
		"device_id": deviceID,
		"event":     event,
	})

	for k, v := range fields {
		entry = entry.WithField(k, v)
	}

	entry.Info("device event")
}

// LogTaskEvent 记录任务事件
func (l *Logger) LogTaskEvent(taskID, deviceID, event string, fields map[string]interface{}) {
	entry := l.WithFields(logrus.Fields{
		"component": "task",
		"task_id":   taskID,
		"device_id": deviceID,
		"event":     event,
	})

	for k, v := range fields {
		entry = entry.WithField(k, v)
	}

	entry.Info("task event")
}

// LogScanProgress 记录扫描进度
func (l *Logger) LogScanProgress(taskID string, progress, totalPages, scannedPages int) {
	l.WithFields(logrus.Fields{
		"component":     "scan",
		"task_id":       taskID,
		"progress":      progress,
		"total_pages":   totalPages,
		"scanned_pages": scannedPages,
	}).Debug("scan progress")
}

// SetLevel 动态设置日志级别
func (l *Logger) SetLevel(level string) error {
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	l.Logger.SetLevel(lvl)
	return nil
}

// WithComponent 添加组件字段
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.WithField("component", component)
}

// WithRequestID 添加请求ID字段
func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithField("request_id", requestID)
}

// WithDeviceID 添加设备ID字段
func (l *Logger) WithDeviceID(deviceID string) *logrus.Entry {
	return l.WithField("device_id", deviceID)
}

// WithTaskID 添加任务ID字段
func (l *Logger) WithTaskID(taskID string) *logrus.Entry {
	return l.WithField("task_id", taskID)
}

// AuditEvent 审计事件类型常量
const (
	AuditEventDeviceCreated   = "device.created"
	AuditEventDeviceUpdated   = "device.updated"
	AuditEventDeviceDeleted   = "device.deleted"
	AuditEventDeviceConnected = "device.connected"
	AuditEventTaskCreated     = "task.created"
	AuditEventTaskStarted     = "task.started"
	AuditEventTaskCompleted   = "task.completed"
	AuditEventTaskFailed      = "task.failed"
	AuditEventTaskCancelled   = "task.cancelled"
	AuditEventFileDownloaded  = "file.downloaded"
	AuditEventFileDeleted     = "file.deleted"
	AuditEventConfigUpdated   = "config.updated"
	AuditEventUserLogin       = "user.login"
	AuditEventUserLogout      = "user.logout"
)
