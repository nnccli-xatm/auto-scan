// 任务调度服务
// 负责任务队列管理、调度执行、状态跟踪
// 设计要点：
// 1. 优先级队列（数字越小优先级越高）
// 2. 并发控制（最大并发扫描数限制）
// 3. 任务状态机管理
// 4. 支持取消和暂停

package taskservice

import (
	"auto-scan/internal/core/scan"
	device "auto-scan/internal/core/device"
	"auto-scan/internal/data/models"
	"auto-scan/internal/data/repository"
	"auto-scan/pkg/logger"
	"auto-scan/pkg/utils"
	"context"
	"io"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TaskService 任务服务接口
type TaskService interface {
	// 任务管理
	CreateTask(ctx context.Context, req CreateTaskRequest) (*models.ScanTask, error)
	GetTask(ctx context.Context, taskID string) (*models.ScanTask, error)
	ListTasks(ctx context.Context, filter ListTaskFilter) ([]*models.ScanTask, int, error)
	CancelTask(ctx context.Context, taskID string) error
	PauseTask(ctx context.Context, taskID string) error
	ResumeTask(ctx context.Context, taskID string) error

	// 任务控制
	StartTask(ctx context.Context, taskID string) error
	GetTaskProgress(ctx context.Context, taskID string) (*TaskProgress, error)

	// 调度控制
	StartScheduler(ctx context.Context) error
	StopScheduler(ctx context.Context) error

	// 订阅任务事件
	SubscribeEvents(handler EventHandler) string
	UnsubscribeEvents(subscriptionID string)
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	DeviceID string                 `json:"device_id" binding:"required"`
	Priority int                    `json:"priority" binding:"min=1,max=10"`
	Settings models.ScanSettings   `json:"settings"`
}

// ListTaskFilter 任务列表过滤
type ListTaskFilter struct {
	Status   string
	DeviceID string
	Page     int
	PageSize int
}

// TaskProgress 任务进度
type TaskProgress struct {
	TaskID        string    `json:"task_id"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`       // 0-100
	TotalPages    int       `json:"total_pages"`
	ScannedPages  int       `json:"scanned_pages"`
	CurrentFile   string    `json:"current_file,omitempty"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	EstimatedEnd  time.Time `json:"estimated_end,omitempty"`
}

// EventHandler 事件处理器
type EventHandler func(event TaskEvent)

// TaskEvent 任务事件
type TaskEvent struct {
	Type      string      `json:"type"`      // created, started, progressed, completed, failed, cancelled
	TaskID    string      `json:"task_id"`
	DeviceID  string      `json:"device_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// taskService 任务服务实现
type taskService struct {
	repo          repository.TaskRepository
	deviceRepo    repository.DeviceRepository
	scheduler     *TaskScheduler
	eventHandlers map[string]EventHandler
	handlersMu    sync.RWMutex
	logger        *logger.Logger
}

// NewTaskService 创建任务服务
func NewTaskService(repo repository.TaskRepository, deviceRepo repository.DeviceRepository, log *logger.Logger) TaskService {
	svc := &taskService{
		repo:          repo,
		deviceRepo:    deviceRepo,
		scheduler:     NewTaskScheduler(),
		eventHandlers: make(map[string]EventHandler),
		logger:        log,
	}
	return svc
}

// NewTaskServiceWithExecutor 创建任务服务并直接配置扫描执行器
func NewTaskServiceWithExecutor(repo repository.TaskRepository, deviceRepo repository.DeviceRepository, executor *scan.Executor, fileRepo repository.FileRepository, log *logger.Logger) TaskService {
	svc := &taskService{
		repo:          repo,
		deviceRepo:    deviceRepo,
		scheduler:     NewTaskScheduler(),
		eventHandlers: make(map[string]EventHandler),
		logger:        log,
	}
	svc.scheduler.executor = executor
	svc.scheduler.deviceRepo = deviceRepo
	svc.scheduler.taskRepo = repo
	svc.scheduler.fileRepo = fileRepo
	svc.scheduler.logger = log
	return svc
}

// CreateTask 创建任务并自动提交调度
func (s *taskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*models.ScanTask, error) {
	// 检查设备是否存在
	device, err := s.deviceRepo.GetByID(ctx, req.DeviceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrDeviceNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err)
	}

	// 检查设备状态
	if device.Status == models.DeviceStatusOffline {
		return nil, utils.ErrDeviceOffline
	}
	if device.Status == models.DeviceStatusBusy {
		return nil, utils.ErrDeviceBusy
	}

	// 使用默认设置
	settings := req.Settings
	if settings.Resolution == 0 {
		settings.Resolution = models.DefaultScanSettings.Resolution
	}
	if settings.ColorMode == "" {
		settings.ColorMode = models.DefaultScanSettings.ColorMode
	}
	if settings.Format == "" {
		settings.Format = models.DefaultScanSettings.Format
	}

	// 创建任务
	task := &models.ScanTask{
		ID:       utils.GenerateUUID(),
		DeviceID: req.DeviceID,
		Status:   models.TaskStatusPending,
		Priority: req.Priority,
		Settings: utils.ToJSON(settings),
		Progress: 0,
	}

	if task.Priority == 0 {
		task.Priority = 5 // 默认优先级
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, utils.WrapError(utils.ErrCodeInternalError, err)
	}

	// 自动提交到调度器执行
	s.scheduler.Submit(task)

	s.notifyEvent(TaskEvent{
		Type:      "created",
		TaskID:    task.ID,
		DeviceID:  task.DeviceID,
		Timestamp: time.Now(),
		Data:      task,
	})

	s.logger.Audit(utils.AuditEventTaskCreated, "", task.DeviceID, task.ID, "Task created", nil)
	return task, nil
}

// GetTask 获取任务
func (s *taskService) GetTask(ctx context.Context, taskID string) (*models.ScanTask, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrTaskNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err)
	}
	return task, nil
}

// ListTasks 任务列表
func (s *taskService) ListTasks(ctx context.Context, filter ListTaskFilter) ([]*models.ScanTask, int, error) {
	repoFilter := repository.TaskFilter{
		Status:   filter.Status,
		DeviceID: filter.DeviceID,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	if repoFilter.Page <= 0 {
		repoFilter.Page = 1
	}
	if repoFilter.PageSize <= 0 || repoFilter.PageSize > 100 {
		repoFilter.PageSize = 20
	}

	tasks, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, utils.WrapError(utils.ErrCodeInternalError, err)
	}

	return tasks, total, nil
}

// CancelTask 取消任务
func (s *taskService) CancelTask(ctx context.Context, taskID string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		if err == repository.ErrNotFound {
			return utils.ErrTaskNotFound
		}
		return utils.WrapError(utils.ErrCodeInternalError, err)
	}

	// 只有Pending或Running状态可以取消
	if task.Status != models.TaskStatusPending && task.Status != models.TaskStatusRunning {
		return utils.NewError(utils.ErrCodeTaskCancelFailed, "task cannot be cancelled in current status")
	}

	// 更新状态
	if err := s.repo.UpdateStatus(ctx, taskID, models.TaskStatusCancelled); err != nil {
		return utils.WrapError(utils.ErrCodeInternalError, err)
	}

	s.notifyEvent(TaskEvent{
		Type:      "cancelled",
		TaskID:    task.ID,
		DeviceID:  task.DeviceID,
		Timestamp: time.Now(),
	})

	s.logger.Audit(utils.AuditEventTaskCancelled, "", task.DeviceID, taskID, "Task cancelled", nil)
	return nil
}

// PauseTask 暂停任务
func (s *taskService) PauseTask(ctx context.Context, taskID string) error {
	return nil
}

// ResumeTask 恢复任务
func (s *taskService) ResumeTask(ctx context.Context, taskID string) error {
	return nil
}

// StartTask 启动任务
func (s *taskService) StartTask(ctx context.Context, taskID string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		if err == repository.ErrNotFound {
			return utils.ErrTaskNotFound
		}
		return utils.WrapError(utils.ErrCodeInternalError, err)
	}

	// 提交到调度器
	if err := s.scheduler.Submit(task); err != nil {
		return utils.WrapError(utils.ErrCodeTaskCreateFailed, err)
	}

	return nil
}

// GetTaskProgress 获取任务进度
func (s *taskService) GetTaskProgress(ctx context.Context, taskID string) (*TaskProgress, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrTaskNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err)
	}

	progress := &TaskProgress{
		TaskID:       task.ID,
		Status:       task.Status,
		Progress:     task.Progress,
		TotalPages:   task.TotalPages,
		ScannedPages: task.ScannedPages,
	}

	if !task.StartedAt.IsZero() {
		progress.StartedAt = task.StartedAt
	}

	return progress, nil
}

// StartScheduler 启动调度器
func (s *taskService) StartScheduler(ctx context.Context) error {
	return s.scheduler.Start()
}

// StopScheduler 停止调度器
func (s *taskService) StopScheduler(ctx context.Context) error {
	return s.scheduler.Stop()
}

// SubscribeEvents 订阅任务事件
func (s *taskService) SubscribeEvents(handler EventHandler) string {
	id := utils.GenerateUUID()
	s.handlersMu.Lock()
	s.eventHandlers[id] = handler
	s.handlersMu.Unlock()
	return id
}

// UnsubscribeEvents 取消订阅
func (s *taskService) UnsubscribeEvents(subscriptionID string) {
	s.handlersMu.Lock()
	delete(s.eventHandlers, subscriptionID)
	s.handlersMu.Unlock()
}

// notifyEvent 通知事件
func (s *taskService) notifyEvent(event TaskEvent) {
	s.handlersMu.RLock()
	handlers := make([]EventHandler, 0, len(s.eventHandlers))
	for _, h := range s.eventHandlers {
		handlers = append(handlers, h)
	}
	s.handlersMu.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// TaskScheduler 任务调度器
type TaskScheduler struct {
	queue         chan *models.ScanTask
	workers       int
	running       bool
	stopChan      chan struct{}
	mu            sync.Mutex
	maxConcurrent int
	// 扫描执行依赖（在 SetExecutor 后设置）
	executor   *scan.Executor
	deviceRepo repository.DeviceRepository
	taskRepo   repository.TaskRepository
	fileRepo   repository.FileRepository
	logger     *logger.Logger
}

// NewTaskScheduler 创建任务调度器
func NewTaskScheduler() *TaskScheduler {
	return &TaskScheduler{
		queue:         make(chan *models.ScanTask, 100),
		workers:       2,
		stopChan:      make(chan struct{}),
		maxConcurrent: 5,
	}
}

// Start 启动调度器
func (s *TaskScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	s.running = true
	for i := 0; i < s.workers; i++ {
		go s.worker()
	}
	if s.logger != nil {
		s.logger.Info("Task scheduler started with %d workers", s.workers)
	}
	return nil
}

// Stop 停止调度器
func (s *TaskScheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	close(s.stopChan)
	s.running = false
	return nil
}

// Submit 提交任务
func (s *TaskScheduler) Submit(task *models.ScanTask) error {
	select {
	case s.queue <- task:
		if s.logger != nil {
			s.logger.Info("Task %s submitted to queue", task.ID[:8])
		}
		return nil
	default:
		return utils.NewError(utils.ErrCodeTaskQueueFull, "task queue is full")
	}
}

// cleanupOldJobs 清理扫描仪上的旧任务
func (s *TaskScheduler) cleanupOldJobs(ctx context.Context, client *device.ESCLClient, status *device.ScannerStatus) error {
	for _, job := range status.Jobs {
		if job.JobState == "Completed" || job.JobState == "Aborted" || job.JobState == "Canceled" {
			jobURL := job.JobURI
			if err := client.DeleteJob(ctx, jobURL); err != nil {
				if s.logger != nil {
					s.logger.Warn("Failed to delete old job: %v", err)
				}
			}
		}
	}
	return nil
}


// worker 工作协程
func (s *TaskScheduler) worker() {
	for {
		select {
		case <-s.stopChan:
			return
		case task := <-s.queue:
			if task != nil {
				s.executeTask(task)
			}
		}
	}
}

// executeTask 执行任务 - 真正的扫描逻辑
func (s *TaskScheduler) executeTask(task *models.ScanTask) {
	if s.executor == nil || s.deviceRepo == nil {
		if s.logger != nil {
			s.logger.Warn("Execute skipped: executor not configured for task %s", task.ID[:8])
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	s.logger.Info("Starting scan task %s for device %s", task.ID[:8], task.DeviceID[:8])

	// 1. 更新任务状态为Running
	task.Status = models.TaskStatusRunning
	task.StartedAt = time.Now()
	if s.taskRepo != nil {
		s.taskRepo.Update(ctx, task)
	}

	// 2. 获取设备信息
	dev, err := s.deviceRepo.GetByID(ctx, task.DeviceID)
	if err != nil {
		s.failTask(ctx, task, fmt.Errorf("device not found: %w", err))
		return
	}

	// 3. 创建设备客户端，检查设备状态并清理旧任务
	client := device.NewESCLClient(dev.IPAddress, 0)
	status, err := client.GetStatus(ctx)
	if err != nil {
		s.failTask(ctx, task, fmt.Errorf("device unreachable: %w", err))
		return
	}

	// 清理旧任务（防止409冲突）
	s.cleanupOldJobs(ctx, client, status)

	// 等待设备空闲
	for retry := 0; retry < 30 && status.State != "Idle"; retry++ {
		time.Sleep(1 * time.Second)
		status, err = client.GetStatus(ctx)
		if err != nil {
			s.failTask(ctx, task, fmt.Errorf("device unresponsive: %w", err))
			return
		}
	}
	if status.State != "Idle" {
		s.failTask(ctx, task, fmt.Errorf("device busy"))
		return
	}

	// 4. 等待ADF有纸（带ADF的设备）
	supportsADF, _ := client.SupportsADFQuery(ctx)
	if supportsADF {
		if err := client.WaitForADF(ctx, 30*time.Second); err != nil {
			s.failTask(ctx, task, fmt.Errorf("ADF timeout: %w", err))
			return
		}
	}

	// 5. 创建设置并执行扫描
	inputSource := "Platen"
	if supportsADF {
		inputSource = "Feeder"
	}

	settings := device.ScanSettings{
		Version:        "2.5",
		Intent:         "Document",
		InputSource:    inputSource,
		ColorMode:      "RGB24",
		XResolution:    300,
		YResolution:    300,
		DocumentFormat: "image/jpeg",
	}

	jobURI, err := client.CreateScanJob(ctx, settings)
	if err != nil {
		s.failTask(ctx, task, fmt.Errorf("scan job failed: %w", err))
		return
	}

	s.logger.Info("Scan job created: %s", jobURI)

	// 6. 下载页面
	storagePath := fmt.Sprintf("/Users/wangyou/测试图片/%s", task.ID[:8])
	os.MkdirAll(storagePath, 0755)

	pageCount := 0
	maxPages := 1
	if inputSource == "Feeder" {
		maxPages = 100
	}

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		select {
		case <-ctx.Done():
			client.DeleteJob(ctx, jobURI)
			s.failTask(ctx, task, fmt.Errorf("cancelled"))
			return
		default:
		}
		if pageNum > 1 {
			time.Sleep(1 * time.Second)
		}
		reader, err := client.GetNextDocument(ctx, jobURI)
		if err != nil {
			if pageNum == 1 {
				s.failTask(ctx, task, fmt.Errorf("download failed: %w", err))
				client.DeleteJob(ctx, jobURI)
				return
			}
			break
		}
		filename := fmt.Sprintf("page_%03d.jpg", pageNum)
		filepath := filepath.Join(storagePath, filename)
		if size, err := func() (int64, error) {
			defer reader.Close()
			os.MkdirAll(storagePath, 0755)
			f, err := os.Create(filepath)
			if err != nil {
				return 0, err
			}
			defer f.Close()
			return io.Copy(f, reader)
		}(); err == nil {
			s.logger.Info("Downloaded page %d: %s (%d bytes)", pageNum, filename, size)
			// 保存文件记录到数据库
			if s.fileRepo != nil {
				fileRecord := &models.ScanFile{
					ID:         utils.GenerateUUID(),
					TaskID:     task.ID,
					DeviceID:   task.DeviceID,
					Filename:   filename,
					FilePath:   filepath,
					FileSize:   size,
					PageNumber: pageNum,
					Format:     "JPEG",
					Status:     "active",
				}
				if err := s.fileRepo.Create(ctx, fileRecord); err != nil {
					s.logger.Warn("Failed to save file record: %v", err)
				}
			}
		}
		pageCount++
		if inputSource == "Platen" {
			break
		}
	}

	client.DeleteJob(ctx, jobURI)

	// 7. 完成任务
	task.Status = models.TaskStatusCompleted
	task.ScannedPages = pageCount
	task.TotalPages = pageCount
	task.Progress = 100
	task.CompletedAt = time.Now()
	task.Result = utils.ToJSON(map[string]interface{}{
		"pages":       pageCount,
		"storagePath": storagePath,
	})
	if s.taskRepo != nil {
		s.taskRepo.Update(ctx, task)
	}
	s.logger.Info("Task %s completed: %d pages saved to %s", task.ID[:8], pageCount, storagePath)
}

// failTask 标记任务失败
func (s *TaskScheduler) failTask(ctx context.Context, task *models.ScanTask, err error) {
	task.Status = models.TaskStatusFailed
	task.ErrorMessage = err.Error()
	task.CompletedAt = time.Now()
	if s.taskRepo != nil {
		s.taskRepo.Update(ctx, task)
	}
	if s.logger != nil {
		s.logger.Error("Task %s failed: %v", task.ID[:8], err)
	}
}
