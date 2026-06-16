package handlers

import (
	"auto-scan/internal/data/models"
	"auto-scan/internal/service/deviceservice"
	"auto-scan/internal/service/systemservice"
	"auto-scan/internal/service/taskservice"
	"auto-scan/pkg/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler 统一Handler容器
type Handler struct {
	DeviceService deviceservice.DeviceService
	TaskService   taskservice.TaskService
	SystemService systemservice.SystemService
}

// NewHandler 创建Handler容器
func NewHandler(ds deviceservice.DeviceService, ts taskservice.TaskService, ss systemservice.SystemService) *Handler {
	return &Handler{
		DeviceService: ds,
		TaskService:   ts,
		SystemService: ss,
	}
}

// ==================== 设备管理 ====================

// ListDevices 获取设备列表
func (h *Handler) ListDevices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	vendor := c.Query("vendor")
	protocol := c.Query("protocol")

	filter := deviceservice.ListDeviceFilter{
		Status:   status,
		Vendor:   vendor,
		Protocol: protocol,
		Page:     page,
		PageSize: pageSize,
	}

	devices, total, err := h.DeviceService.ListDevices(c.Request.Context(), filter)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.PaginationSuccess(c, devices, page, pageSize, total)
}

// GetDevice 获取设备详情
func (h *Handler) GetDevice(c *gin.Context) {
	deviceID := c.Param("id")

	dev, err := h.DeviceService.GetDevice(c.Request.Context(), deviceID)
	if err != nil {
		utils.NotFound(c, "device")
		return
	}

	utils.Success(c, dev)
}

// CreateDevice 创建设备
func (h *Handler) CreateDevice(c *gin.Context) {
	var req deviceservice.AddDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	dev, err := h.DeviceService.AddDevice(c.Request.Context(), req)
	if err != nil {
		utils.ErrorWithCode(c, utils.HTTPStatusCode(utils.ErrCodeDeviceExists), err.Error())
		return
	}

	utils.Created(c, dev)
}

// UpdateDevice 更新设备
func (h *Handler) UpdateDevice(c *gin.Context) {
	deviceID := c.Param("id")

	var req deviceservice.UpdateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	dev, err := h.DeviceService.UpdateDevice(c.Request.Context(), deviceID, req)
	if err != nil {
		utils.NotFound(c, "device")
		return
	}

	utils.Success(c, dev)
}

// DeleteDevice 删除设备
func (h *Handler) DeleteDevice(c *gin.Context) {
	deviceID := c.Param("id")

	if err := h.DeviceService.DeleteDevice(c.Request.Context(), deviceID); err != nil {
		utils.NotFound(c, "device")
		return
	}

	utils.NoContent(c)
}

// DiscoverDevices 发现设备
func (h *Handler) DiscoverDevices(c *gin.Context) {
	devices, err := h.DeviceService.DiscoverDevices(c.Request.Context())
	if err != nil {
		utils.InternalError(c, "discovery failed: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"found":   len(devices),
		"devices": devices,
	})
}

// GetDeviceStatus 获取设备状态
func (h *Handler) GetDeviceStatus(c *gin.Context) {
	deviceID := c.Param("id")

	status, err := h.DeviceService.GetDeviceStatus(c.Request.Context(), deviceID)
	if err != nil {
		utils.NotFound(c, "device")
		return
	}

	utils.Success(c, status)
}

// ==================== 任务管理 ====================

// ListTasks 获取任务列表
func (h *Handler) ListTasks(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := taskservice.ListTaskFilter{
		Status:   c.Query("status"),
		DeviceID: c.Query("device_id"),
		Page:     page,
		PageSize: pageSize,
	}

	tasks, total, err := h.TaskService.ListTasks(c.Request.Context(), filter)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.PaginationSuccess(c, tasks, page, pageSize, total)
}

// CreateTask 创建扫描任务
func (h *Handler) CreateTask(c *gin.Context) {
	var req taskservice.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	task, err := h.TaskService.CreateTask(c.Request.Context(), req)
	if err != nil {
		utils.ErrorWithCode(c, utils.HTTPStatusCode(utils.ErrCodeTaskCreateFailed), err.Error())
		return
	}

	utils.Created(c, task)
}

// GetTask 获取任务详情
func (h *Handler) GetTask(c *gin.Context) {
	taskID := c.Param("id")

	task, err := h.TaskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		utils.NotFound(c, "task")
		return
	}

	utils.Success(c, task)
}

// CancelTask 取消任务
func (h *Handler) CancelTask(c *gin.Context) {
	taskID := c.Param("id")

	if err := h.TaskService.CancelTask(c.Request.Context(), taskID); err != nil {
		utils.ErrorWithCode(c, utils.HTTPStatusCode(utils.ErrCodeTaskCancelFailed), err.Error())
		return
	}

	utils.Success(c, gin.H{"id": taskID, "status": "cancelled"})
}

// GetTaskProgress 获取任务进度
func (h *Handler) GetTaskProgress(c *gin.Context) {
	taskID := c.Param("id")

	progress, err := h.TaskService.GetTaskProgress(c.Request.Context(), taskID)
	if err != nil {
		utils.NotFound(c, "task")
		return
	}

	utils.Success(c, progress)
}

// ==================== 系统管理 ====================

// GetSystemStatus 获取系统状态
func (h *Handler) GetSystemStatus(c *gin.Context) {
	status, err := h.SystemService.GetSystemStatus(c.Request.Context())
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, status)
}

// GetSystemLogs 获取审计日志
func (h *Handler) GetSystemLogs(c *gin.Context) {
	filter := systemservice.LogFilter{
		Level:    c.Query("level"),
		Page:     1,
		PageSize: 50,
	}

	logs, total, err := h.SystemService.GetLogs(c.Request.Context(), filter)
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.PaginationSuccess(c, logs, 1, 50, total)
}

// GetSystemConfig 获取系统配置
func (h *Handler) GetSystemConfig(c *gin.Context) {
	cfg, err := h.SystemService.GetConfig(c.Request.Context())
	if err != nil {
		utils.InternalError(c, err.Error())
		return
	}
	utils.Success(c, cfg)
}

// UpdateSystemConfig 更新系统配置
func (h *Handler) UpdateSystemConfig(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.SystemService.UpdateConfig(c.Request.Context(), updates); err != nil {
		utils.InternalError(c, err.Error())
		return
	}

	utils.Success(c, gin.H{"updated": true})
}

// ==================== 文件管理 ====================

// ListFiles 获取文件列表
func (h *Handler) ListFiles(c *gin.Context) {
	utils.PaginationSuccess(c, []models.ScanFile{}, 1, 20, 0)
}

// GetFile 获取文件详情
func (h *Handler) GetFile(c *gin.Context) {
	utils.NotFound(c, "file")
}

// DeleteFile 删除文件
func (h *Handler) DeleteFile(c *gin.Context) {
	utils.NoContent(c)
}

// DownloadFile 下载文件
func (h *Handler) DownloadFile(c *gin.Context) {
	fileID := c.Param("id")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.jpg\"", fileID))
	c.Data(http.StatusOK, "image/jpeg", []byte{})
}

// BatchDownloadFiles 批量下载
func (h *Handler) BatchDownloadFiles(c *gin.Context) {
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename=\"scans.zip\"")
	c.Data(http.StatusOK, "application/zip", []byte{})
}

// BatchDeleteFiles 批量删除
func (h *Handler) BatchDeleteFiles(c *gin.Context) {
	utils.Success(c, gin.H{"deleted": 0})
}
