// 统一响应格式
// 提供标准化的API响应结构

package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginationData 分页数据结构
type PaginationData struct {
	List       interface{} `json:"list"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination 分页信息
type Pagination struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrCodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    ErrCodeSuccess,
		Message: message,
		Data:    data,
	})
}

// Created 创建成功响应
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    ErrCodeSuccess,
		Message: "created",
		Data:    data,
	})
}

// NoContent 无内容响应
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error 错误响应
func Error(c *gin.Context, err *AppError) {
	statusCode := HTTPStatusCode(err.Code)
	c.JSON(statusCode, Response{
		Code:    err.Code,
		Message: err.Message,
		Data:    gin.H{"details": err.Details},
	})
}

// ErrorWithCode 错误响应（自定义错误码）
func ErrorWithCode(c *gin.Context, code int, message string) {
	if message == "" {
		message = GetErrorMessage(code)
	}
	statusCode := HTTPStatusCode(code)
	c.JSON(statusCode, Response{
		Code:    code,
		Message: message,
	})
}

// BadRequest 错误请求响应
func BadRequest(c *gin.Context, message string) {
	if message == "" {
		message = GetErrorMessage(ErrCodeInvalidRequest)
	}
	c.JSON(http.StatusBadRequest, Response{
		Code:    ErrCodeInvalidRequest,
		Message: message,
	})
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = GetErrorMessage(ErrCodeUnauthorized)
	}
	c.JSON(http.StatusUnauthorized, Response{
		Code:    ErrCodeUnauthorized,
		Message: message,
	})
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = GetErrorMessage(ErrCodeForbidden)
	}
	c.JSON(http.StatusForbidden, Response{
		Code:    ErrCodeForbidden,
		Message: message,
	})
}

// NotFound 资源不存在响应
func NotFound(c *gin.Context, resource string) {
	message := GetErrorMessage(ErrCodeNotFound)
	if resource != "" {
		message = resource + " not found"
	}
	c.JSON(http.StatusNotFound, Response{
		Code:    ErrCodeNotFound,
		Message: message,
	})
}

// PaginationSuccess 分页成功响应
func PaginationSuccess(c *gin.Context, list interface{}, page, pageSize, total int) {
	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, Response{
		Code:    ErrCodeSuccess,
		Message: "success",
		Data: PaginationData{
			List: list,
			Pagination: Pagination{
				Page:       page,
				PageSize:   pageSize,
				Total:      total,
				TotalPages: totalPages,
			},
		},
	})
}

// ValidationError 参数校验错误响应
func ValidationError(c *gin.Context, errors map[string]string) {
	c.JSON(http.StatusUnprocessableEntity, Response{
		Code:    ErrCodeValidationFailed,
		Message: "validation failed",
		Data:    gin.H{"errors": errors},
	})
}

// InternalError 服务器内部错误响应
func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = GetErrorMessage(ErrCodeInternalError)
	}
	c.JSON(http.StatusInternalServerError, Response{
		Code:    ErrCodeInternalError,
		Message: message,
	})
}