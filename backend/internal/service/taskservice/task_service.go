// 任务调度服务
// 负责任务队列管理、调度执行、状态跟踪
// 设计要点：
// 1. 优先级队列（数字越小优先级越高）
// 2. 并发控制（最大并发扫描数限制）
// 3. 任务状态机管理
// 4. 支持取消和暂停

package taskservice

import (
	"auto-scan/internal/data/models"
	"auto-scan/internal/data/repository"
	"auto-scan/pkg/logger"
	"auto-scan/pkg/utils"
	"context"
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
	return &taskService{
		repo:          repo,
		deviceRepo:    deviceRepo,
		scheduler:     NewTaskScheduler(),
		eventHandlers: make(map[string]EventHandler),
		logger:        log,
	}
}

// CreateTask 创建任务
func (s *taskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*models.ScanTask, error) {
	// 检查设备是否存在
	device, err := s.deviceRepo.GetByID(ctx, req.DeviceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrDeviceNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to get device")
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
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to create task")
	}

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
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to get task")
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
		return nil, 0, utils.WrapError(utils.ErrCodeInternalError, err, "failed to list tasks")
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
		return utils.WrapError(utils.ErrCodeInternalError, err, "failed to get task")
	}

	// 只有Pending或Running状态可以取消
	if task.Status != models.TaskStatusPending && task.Status != models.TaskStatusRunning {
		return utils.NewError(utils.ErrCodeTaskCancelFailed, "task cannot be cancelled in current status")
	}

	// 更新状态
	if err := s.repo.UpdateStatus(ctx, taskID, models.TaskStatusCancelled); err != nil {
		return utils.WrapError(utils.ErrCodeInternalError, err, "failed to cancel task")
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
	// TODO: 实现暂停逻辑
	return nil
}

// ResumeTask 恢复任务
func (s *taskService) ResumeTask(ctx context.Context, taskID string) error {
	// TODO: 实现恢复逻辑
	return nil
}

// StartTask 启动任务
func (s *taskService) StartTask(ctx context.Context, taskID string) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		if err == repository.ErrNotFound {
			return utils.ErrTaskNotFound
		}
		return utils.WrapError(utils.ErrCodeInternalError, err, "failed to get task")
	}

	// 提交到调度器
	if err := s.scheduler.Submit(task); err != nil {
		return utils.WrapError(utils.ErrCodeTaskCreateFailed, err, "failed to submit task to scheduler")
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
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to get task")
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
	queue       chan *models.ScanTask
	workers     int
	running     bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	maxConcurrent int
}

// NewTaskScheduler 创建任务调度器
func NewTaskScheduler() *TaskScheduler {
	return &TaskScheduler{
		queue:         make(chan *models.ScanTask, 100),
		workers:       5,
		stopChan:      make(chan struct{}),
		maxConcurrent: 5,
	}
}

// Start 启动调度器
func (s *TaskScheduler) Start() error {
	if s.running {
		return nil
	}

	s.running = true
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker()
	}

	return nil
}

// Stop 停止调度器
func (s *TaskScheduler) Stop() error {
	if !s.running {
		return nil
	}

	close(s.stopChan)
	s.wg.Wait()
	s.running = false
	return nil
}

// Submit 提交任务
func (s *TaskScheduler) Submit(task *models.ScanTask) error {
	select {
	case s.queue <- task:
		return nil
	default:
		return utils.NewError(utils.ErrCodeTaskQueueFull, "task queue is full")
	}
}

// worker 工作协程
func (s *TaskScheduler) worker() {
	defer s.wg.Done()

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

// executeTask 执行任务
func (s *TaskScheduler) executeTask(task *models.ScanTask) {
	// TODO: 实现具体的任务执行逻辑
	// 1. 更新任务状态为Running
	// 2. 创建设备客户端
	// 3. 执行扫描流程
	// 4. 更新任务状态和结果
}

// GenerateUUID 生成UUID（工具函数）
func GenerateUUID() string {
	return utils.GenerateUUID()
}