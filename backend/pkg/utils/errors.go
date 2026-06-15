// 错误处理
// 统一错误码定义和错误封装

package utils

import (
	"errors"
	"fmt"
)

// 错误码定义
const (
	// 通用错误 0-999
	ErrCodeSuccess           = 0
	ErrCodeUnknown           = 1
	ErrCodeInvalidRequest    = 400001
	ErrCodeUnauthorized      = 401001
	ErrCodeForbidden         = 403001
	ErrCodeNotFound          = 404001
	ErrCodeMethodNotAllowed  = 405001
	ErrCodeConflict          = 409001
	ErrCodeValidationFailed  = 422001
	ErrCodeTooManyRequests   = 429001
	ErrCodeInternalError     = 500001
	ErrCodeServiceUnavailable = 503001

	// 设备相关错误 1000-1999
	ErrCodeDeviceNotFound     = 1001001
	ErrCodeDeviceOffline      = 1001002
	ErrCodeDeviceBusy         = 1001003
	ErrCodeDeviceError        = 1001004
	ErrCodeDeviceExists       = 1001005
	ErrCodeDeviceConnectFailed = 1001006
	ErrCodeDeviceNotSupported = 1001007

	// 任务相关错误 2000-2999
	ErrCodeTaskNotFound       = 1002001
	ErrCodeTaskCreateFailed   = 1002002
	ErrCodeTaskCancelFailed   = 1002003
	ErrCodeTaskAlreadyRunning = 1002004
	ErrCodeTaskQueueFull      = 1002005

	// 文件相关错误 3000-3999
	ErrCodeFileNotFound       = 1003001
	ErrCodeFileDownloadFailed = 1003002
	ErrCodeFileDeleteFailed   = 1003003
	ErrCodeFileUploadFailed   = 1003004
	ErrCodeStorageFull        = 1003005
)

// 错误码映射表
var errorCodeMessages = map[int]string{
	ErrCodeSuccess:           "success",
	ErrCodeUnknown:           "unknown error",
	ErrCodeInvalidRequest:    "invalid request",
	ErrCodeUnauthorized:      "unauthorized",
	ErrCodeForbidden:         "forbidden",
	ErrCodeNotFound:          "resource not found",
	ErrCodeMethodNotAllowed:  "method not allowed",
	ErrCodeConflict:          "resource conflict",
	ErrCodeValidationFailed:  "validation failed",
	ErrCodeTooManyRequests:   "too many requests",
	ErrCodeInternalError:     "internal server error",
	ErrCodeServiceUnavailable: "service unavailable",

	ErrCodeDeviceNotFound:      "device not found",
	ErrCodeDeviceOffline:       "device is offline",
	ErrCodeDeviceBusy:          "device is busy",
	ErrCodeDeviceError:         "device error",
	ErrCodeDeviceExists:        "device already exists",
	ErrCodeDeviceConnectFailed: "failed to connect device",
	ErrCodeDeviceNotSupported:  "device not supported",

	ErrCodeTaskNotFound:       "task not found",
	ErrCodeTaskCreateFailed:   "failed to create task",
	ErrCodeTaskCancelFailed:   "failed to cancel task",
	ErrCodeTaskAlreadyRunning: "task is already running",
	ErrCodeTaskQueueFull:      "task queue is full",

	ErrCodeFileNotFound:       "file not found",
	ErrCodeFileDownloadFailed: "failed to download file",
	ErrCodeFileDeleteFailed:   "failed to delete file",
	ErrCodeFileUploadFailed:   "failed to upload file",
	ErrCodeStorageFull:        "storage is full",
}

// GetErrorMessage 获取错误码对应的默认消息
func GetErrorMessage(code int) string {
	if msg, ok := errorCodeMessages[code]; ok {
		return msg
	}
	return "unknown error"
}

// AppError 应用错误
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Err     error  `json:"-"`
}

// Error 实现error接口
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewError 创建新的应用错误
func NewError(code int, message string) *AppError {
	if message == "" {
		message = GetErrorMessage(code)
	}
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewErrorf 创建格式化的应用错误
func NewErrorf(code int, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// WrapError 包装原始错误
func WrapError(code int, err error) *AppError {
	if err == nil {
		return nil
	}

	// 如果已经是AppError，直接返回
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	return &AppError{
		Code:    code,
		Message: GetErrorMessage(code),
		Err:     err,
	}
}

// WrapErrorf 包装并格式化错误
func WrapErrorf(code int, err error, format string, args ...interface{}) *AppError {
	if err == nil {
		return nil
	}

	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

// WithDetails 添加错误详情
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// Is 判断错误类型
func (e *AppError) Is(target error) bool {
	if target == nil {
		return false
	}

	if appErr, ok := target.(*AppError); ok {
		return e.Code == appErr.Code
	}

	return errors.Is(e.Err, target)
}

// 预定义错误
var (
	ErrDeviceNotFound     = NewError(ErrCodeDeviceNotFound, "")
	ErrDeviceOffline      = NewError(ErrCodeDeviceOffline, "")
	ErrDeviceBusy         = NewError(ErrCodeDeviceBusy, "")
	ErrTaskNotFound       = NewError(ErrCodeTaskNotFound, "")
	ErrFileNotFound       = NewError(ErrCodeFileNotFound, "")
	ErrUnauthorized       = NewError(ErrCodeUnauthorized, "")
	ErrForbidden          = NewError(ErrCodeForbidden, "")
	ErrValidationFailed   = NewError(ErrCodeValidationFailed, "")
	ErrTooManyRequests    = NewError(ErrCodeTooManyRequests, "")
	ErrInternalError      = NewError(ErrCodeInternalError, "")
	ErrServiceUnavailable = NewError(ErrCodeServiceUnavailable, "")
)

// HTTPStatusCode 获取错误对应的HTTP状态码
func HTTPStatusCode(code int) int {
	switch {
	case code == ErrCodeSuccess:
		return 200
	case code >= 400000 && code < 401000:
		return 400
	case code >= 401000 && code < 403000:
		return 401
	case code >= 403000 && code < 404000:
		return 403
	case code >= 404000 && code < 405000:
		return 404
	case code >= 405000 && code < 409000:
		return 405
	case code >= 409000 && code < 422000:
		return 409
	case code >= 422000 && code < 429000:
		return 422
	case code >= 429000 && code < 500000:
		return 429
	case code >= 500000 && code < 503000:
		return 500
	case code >= 503000:
		return 503
	default:
		return 500
	}
}
