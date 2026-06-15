// 设备管理服务
// 实现设备发现、注册、状态监控、连接管理等核心业务逻辑
// 设计要点：
// 1. 支持多协议设备管理（eSCL、WSD）
// 2. 并发安全的设备状态管理
// 3. 实时状态监控和事件通知
// 4. 设备连接池管理

package deviceservice

import (
	"auto-scan/internal/core/device"
	"auto-scan/internal/data/models"
	"auto-scan/internal/data/repository"
	"auto-scan/pkg/logger"
	"auto-scan/pkg/utils"
	"context"
	"sync"
	"time"
)

// DeviceService 设备服务接口
type DeviceService interface {
	// 设备发现
	DiscoverDevices(ctx context.Context) ([]*models.Device, error)
	AddDevice(ctx context.Context, req AddDeviceRequest) (*models.Device, error)

	// 设备CRUD
	GetDevice(ctx context.Context, deviceID string) (*models.Device, error)
	UpdateDevice(ctx context.Context, deviceID string, req UpdateDeviceRequest) (*models.Device, error)
	DeleteDevice(ctx context.Context, deviceID string) error
	ListDevices(ctx context.Context, filter ListDeviceFilter) ([]*models.Device, int, error)

	// 设备控制
	ConnectDevice(ctx context.Context, deviceID string) error
	DisconnectDevice(ctx context.Context, deviceID string) error
	GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error)

	// 状态监控
	StartMonitoring(ctx context.Context) error
	StopMonitoring(ctx context.Context) error

	// 订阅通知
	SubscribeEvents(handler EventHandler) string
	UnsubscribeEvents(subscriptionID string)
}

// AddDeviceRequest 添加设备请求
type AddDeviceRequest struct {
	Name      string `json:"name" binding:"required,max=100"`
	IPAddress string `json:"ip_address" binding:"required,ip"`
	Protocol  string `json:"protocol" binding:"required,oneof=escl wsd"`
}

// UpdateDeviceRequest 更新设备请求
type UpdateDeviceRequest struct {
	Name   string                 `json:"name,omitempty" binding:"omitempty,max=100"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// ListDeviceFilter 设备列表过滤
type ListDeviceFilter struct {
	Status   string
	Vendor   string
	Protocol string
	Page     int
	PageSize int
}

// DeviceStatus 设备状态
type DeviceStatus struct {
	DeviceID      string    `json:"device_id"`
	Status        string    `json:"status"`         // online, offline, busy, error
	AdfStatus     string    `json:"adf_status"`     // empty, loaded
	ScannerState  string    `json:"scanner_state"`  // idle, processing
	CurrentTask   string    `json:"current_task,omitempty"`
	Capabilities  *models.DeviceCapabilities `json:"capabilities,omitempty"`
	LastSeen      time.Time `json:"last_seen"`
}

// EventHandler 事件处理器
type EventHandler func(event DeviceEvent)

// DeviceEvent 设备事件
type DeviceEvent struct {
	Type      string    `json:"type"`      // discovered, connected, disconnected, status_changed, error
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// deviceService 设备服务实现
type deviceService struct {
	repo          repository.DeviceRepository
	discovery     *device.DiscoveryService
	clients       map[string]*device.ESCLClient  // deviceID -> client
	clientsMu     sync.RWMutex
	monitors      map[string]context.CancelFunc  // deviceID -> cancel func
	monitorsMu    sync.Mutex
	eventHandlers map[string]EventHandler
	handlersMu    sync.RWMutex
	logger        *logger.Logger
}

// NewDeviceService 创建设备服务
func NewDeviceService(repo repository.DeviceRepository, log *logger.Logger) (DeviceService, error) {
	discovery, err := device.NewDiscoveryService()
	if err != nil {
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to create discovery service")
	}

	return &deviceService{
		repo:          repo,
		discovery:     discovery,
		clients:       make(map[string]*device.ESCLClient),
		monitors:      make(map[string]context.CancelFunc),
		eventHandlers: make(map[string]EventHandler),
		logger:        log,
	}, nil
}

// DiscoverDevices 发现设备
func (s *deviceService) DiscoverDevices(ctx context.Context) ([]*models.Device, error) {
	// 启动发现服务
	if err := s.discovery.Start(); err != nil {
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to start discovery")
	}
	defer s.discovery.Stop()

	// 执行一次发现（5秒超时）
	discovered, err := s.discovery.DiscoverOnce(ctx, 5*time.Second)
	if err != nil {
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "discovery failed")
	}

	// 转换为Device模型
	devices := make([]*models.Device, 0, len(discovered))
	for _, d := range discovered {
		dev := &models.Device{
			ID:        utils.GenerateUUID(),
			Name:      d.Name,
			IPAddress: d.IP,
			Protocol:  d.Protocol,
			Model:     d.Model,
			Vendor:    d.Vendor,
			Status:    models.DeviceStatusOnline,
		}

		// 尝试获取设备能力
		client := device.NewESCLClient(d.IP, d.Port)
		caps, err := client.GetCapabilities(ctx)
		if err == nil {
			dev.Capabilities = utils.ToJSON(caps)
		}

		devices = append(devices, dev)
	}

	s.logger.Info("Discovered %d devices", len(devices))
	return devices, nil
}

// AddDevice 添加设备
func (s *deviceService) AddDevice(ctx context.Context, req AddDeviceRequest) (*models.Device, error) {
	// 检查IP是否已存在
	existing, err := s.repo.GetByIPAddress(ctx, req.IPAddress)
	if err == nil && existing != nil {
		return nil, utils.ErrDeviceExists
	}

	// 创建设备模型
	dev := &models.Device{
		ID:        utils.GenerateUUID(),
		Name:      req.Name,
		IPAddress: req.IPAddress,
		Protocol:  req.Protocol,
		Status:    models.DeviceStatusOffline,
	}

	// 尝试连接并获取信息
	client := device.NewESCLClient(req.IPAddress, 0)
	caps, err := client.GetCapabilities(ctx)
	if err == nil {
		dev.Status = models.DeviceStatusOnline
		dev.Capabilities = utils.ToJSON(caps)
		dev.Model = caps.MakeAndModel

		// 解析厂商
		for _, v := range []string{"HP", "Canon", "Ricoh", "Fujitsu", "Brother", "Epson"} {
			if utils.ContainsString(caps.MakeAndModel, v) {
				dev.Vendor = v
				break
			}
		}
	}

	// 保存到数据库
	if err := s.repo.Create(ctx, dev); err != nil {
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to create device")
	}

	s.logger.Audit(utils.AuditEventDeviceCreated, "", dev.ID, "", "Device created", nil)
	return dev, nil
}

// GetDevice 获取设备
func (s *deviceService) GetDevice(ctx context.Context, deviceID string) (*models.Device, error) {
	dev, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrDeviceNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to get device")
	}
	return dev, nil
}

// UpdateDevice 更新设备
func (s *deviceService) UpdateDevice(ctx context.Context, deviceID string, req UpdateDeviceRequest) (*models.Device, error) {
	dev, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrDeviceNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to get device")
	}

	// 更新字段
	if req.Name != "" {
		dev.Name = req.Name
	}
	if req.Config != nil {
		dev.Config = utils.ToJSON(req.Config)
	}

	if err := s.repo.Update(ctx, dev); err != nil {
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to update device")
	}

	s.logger.Audit(utils.AuditEventDeviceUpdated, "", deviceID, "", "Device updated", nil)
	return dev, nil
}

// DeleteDevice 删除设备
func (s *deviceService) DeleteDevice(ctx context.Context, deviceID string) error {
	// 检查设备是否存在
	if _, err := s.repo.GetByID(ctx, deviceID); err != nil {
		if err == repository.ErrNotFound {
			return utils.ErrDeviceNotFound
		}
		return utils.WrapError(utils.ErrCodeInternalError, err, "failed to get device")
	}

	// 断开连接
	s.DisconnectDevice(ctx, deviceID)

	// 删除设备
	if err := s.repo.Delete(ctx, deviceID); err != nil {
		return utils.WrapError(utils.ErrCodeInternalError, err, "failed to delete device")
	}

	s.logger.Audit(utils.AuditEventDeviceDeleted, "", deviceID, "", "Device deleted", nil)
	return nil
}

// ListDevices 设备列表
func (s *deviceService) ListDevices(ctx context.Context, filter ListDeviceFilter) ([]*models.Device, int, error) {
	repoFilter := repository.DeviceFilter{
		Status:   filter.Status,
		Vendor:   filter.Vendor,
		Protocol: filter.Protocol,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}

	if repoFilter.Page <= 0 {
		repoFilter.Page = 1
	}
	if repoFilter.PageSize <= 0 || repoFilter.PageSize > 100 {
		repoFilter.PageSize = 20
	}

	devices, total, err := s.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, 0, utils.WrapError(utils.ErrCodeInternalError, err, "failed to list devices")
	}

	return devices, total, nil
}

// ConnectDevice 连接设备
func (s *deviceService) ConnectDevice(ctx context.Context, deviceID string) error {
	dev, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return utils.ErrDeviceNotFound
		}
		return utils.WrapError(utils.ErrCodeInternalError, err, "failed to get device")
	}

	// 创建客户端
	client := device.NewESCLClient(dev.IPAddress, 0)

	// 测试连接
	_, err = client.GetCapabilities(ctx)
	if err != nil {
		return utils.WrapError(utils.ErrCodeDeviceConnectFailed, err, "failed to connect device")
	}

	// 保存客户端
	s.clientsMu.Lock()
	s.clients[deviceID] = client
	s.clientsMu.Unlock()

	// 更新状态
	if err := s.repo.UpdateStatus(ctx, deviceID, models.DeviceStatusOnline); err != nil {
		s.logger.Error("Failed to update device status: %v", err)
	}

	s.logger.Audit(utils.AuditEventDeviceConnected, "", deviceID, "", "Device connected", nil)
	return nil
}

// DisconnectDevice 断开设备
func (s *deviceService) DisconnectDevice(ctx context.Context, deviceID string) error {
	s.clientsMu.Lock()
	delete(s.clients, deviceID)
	s.clientsMu.Unlock()

	// 停止监控
	s.monitorsMu.Lock()
	if cancel, ok := s.monitors[deviceID]; ok {
		cancel()
		delete(s.monitors, deviceID)
	}
	s.monitorsMu.Unlock()

	// 更新状态
	if err := s.repo.UpdateStatus(ctx, deviceID, models.DeviceStatusOffline); err != nil {
		s.logger.Error("Failed to update device status: %v", err)
	}

	return nil
}

// GetDeviceStatus 获取设备状态
func (s *deviceService) GetDeviceStatus(ctx context.Context, deviceID string) (*DeviceStatus, error) {
	dev, err := s.repo.GetByID(ctx, deviceID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, utils.ErrDeviceNotFound
		}
		return nil, utils.WrapError(utils.ErrCodeInternalError, err, "failed to get device")
	}

	status := &DeviceStatus{
		DeviceID: dev.ID,
		Status:   dev.Status,
		LastSeen: dev.LastSeen,
	}

	// 如果有连接，获取实时状态
	s.clientsMu.RLock()
	client, ok := s.clients[deviceID]
	s.clientsMu.RUnlock()

	if ok && client != nil {
		// 尝试获取设备状态
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		devStatus, err := client.GetStatus(ctx)
		if err == nil {
			status.ScannerState = devStatus.State
			status.AdfStatus = devStatus.AdfState
		}
	}

	return status, nil
}

// StartMonitoring 启动设备监控
func (s *deviceService) StartMonitoring(ctx context.Context) error {
	// TODO: 实现全局监控
	return nil
}

// StopMonitoring 停止设备监控
func (s *deviceService) StopMonitoring(ctx context.Context) error {
	s.monitorsMu.Lock()
	for _, cancel := range s.monitors {
		cancel()
	}
	s.monitors = make(map[string]context.CancelFunc)
	s.monitorsMu.Unlock()
	return nil
}

// SubscribeEvents 订阅设备事件
func (s *deviceService) SubscribeEvents(handler EventHandler) string {
	id := utils.GenerateUUID()
	s.handlersMu.Lock()
	s.eventHandlers[id] = handler
	s.handlersMu.Unlock()
	return id
}

// UnsubscribeEvents 取消订阅
func (s *deviceService) UnsubscribeEvents(subscriptionID string) {
	s.handlersMu.Lock()
	delete(s.eventHandlers, subscriptionID)
	s.handlersMu.Unlock()
}

// notifyEvent 通知事件
func (s *deviceService) notifyEvent(event DeviceEvent) {
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