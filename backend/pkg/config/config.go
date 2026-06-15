// 配置管理系统
// 支持：配置文件(YAML/JSON)、环境变量、命令行参数
// 特性：热重载、配置校验、默认值

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server" json:"server"`
	Database DatabaseConfig `yaml:"database" json:"database"`
	Storage  StorageConfig  `yaml:"storage" json:"storage"`
	Scan     ScanConfig     `yaml:"scan" json:"scan"`
	Device   DeviceConfig   `yaml:"device" json:"device"`
	Log      LogConfig      `yaml:"log" json:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int           `yaml:"port" json:"port"`
	Host         string        `yaml:"host" json:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path           string        `yaml:"path" json:"path"`
	MaxOpenConns   int           `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns   int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Path           string `yaml:"path" json:"path"`
	MaxSize        int64  `yaml:"max_size" json:"max_size"` // 字节
	BackupEnabled  bool   `yaml:"backup_enabled" json:"backup_enabled"`
	BackupPath     string `yaml:"backup_path" json:"backup_path"`
}

// ScanConfig 扫描配置
type ScanConfig struct {
	DefaultResolution int           `yaml:"default_resolution" json:"default_resolution"`
	DefaultColorMode  string        `yaml:"default_color_mode" json:"default_color_mode"`
	MaxConcurrent     int           `yaml:"max_concurrent" json:"max_concurrent"`
	MaxRetries        int           `yaml:"max_retries" json:"max_retries"`
	Timeout           time.Duration `yaml:"timeout" json:"timeout"`
}

// DeviceConfig 设备配置
type DeviceConfig struct {
	MonitorInterval   time.Duration `yaml:"monitor_interval" json:"monitor_interval"`
	AutoDiscover      bool          `yaml:"auto_discover" json:"auto_discover"`
	DiscoverInterval  time.Duration `yaml:"discover_interval" json:"discover_interval"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" json:"level"`           // debug, info, warning, error
	Format     string `yaml:"format" json:"format"`         // json, text
	Output     string `yaml:"output" json:"output"`         // stdout, file
	FilePath   string `yaml:"file_path" json:"file_path"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`     // MB
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`       // days
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Path:            "./data/auto-scan.db",
			MaxOpenConns:    1,
			MaxIdleConns:    1,
			ConnMaxLifetime: time.Hour,
		},
		Storage: StorageConfig{
			Path:          "./data/scans",
			MaxSize:       10 * 1024 * 1024 * 1024, // 10GB
			BackupEnabled: false,
			BackupPath:    "",
		},
		Scan: ScanConfig{
			DefaultResolution: 300,
			DefaultColorMode:  "Color",
			MaxConcurrent:     5,
			MaxRetries:        3,
			Timeout:           5 * time.Minute,
		},
		Device: DeviceConfig{
			MonitorInterval:  2 * time.Second,
			AutoDiscover:     true,
			DiscoverInterval: 5 * time.Minute,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			FilePath:   "./logs/auto-scan.log",
			MaxSize:    100, // 100MB
			MaxBackups: 3,
			MaxAge:     7, // 7 days
		},
	}
}

// Manager 配置管理器
type Manager struct {
	config     *Config
	configPath string
	mu         sync.RWMutex
	watcher    *fsnotify.Watcher
	onChange   func(*Config)
}

// NewManager 创建配置管理器
func NewManager(configPath string) (*Manager, error) {
	// 如果配置文件不存在，创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := SaveConfig(configPath, DefaultConfig()); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}

	// 加载配置
	cfg, err := Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &Manager{
		config:     cfg,
		configPath: configPath,
	}, nil
}

// Get 获取当前配置（线程安全）
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// Reload 重新加载配置
func (m *Manager) Reload() error {
	cfg, err := Load(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	m.mu.Lock()
	m.config = cfg
	m.mu.Unlock()

	// 触发回调
	if m.onChange != nil {
		m.onChange(cfg)
	}

	return nil
}

// OnChange 设置配置变更回调
func (m *Manager) OnChange(fn func(*Config)) {
	m.onChange = fn
}

// StartHotReload 启动热重载（监听配置文件变化）
func (m *Manager) StartHotReload() error {
	if m.watcher != nil {
		return fmt.Errorf("hot reload already started")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	if err := watcher.Add(m.configPath); err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	m.watcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					// 配置文件被修改，重新加载
					time.Sleep(100 * time.Millisecond) // 等待写入完成
					if err := m.Reload(); err != nil {
						fmt.Printf("Failed to reload config: %v\n", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("Config watcher error: %v\n", err)
			}
		}
	}()

	return nil
}

// StopHotReload 停止热重载
func (m *Manager) StopHotReload() error {
	if m.watcher == nil {
		return nil
	}

	if err := m.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	m.watcher = nil
	return nil
}

// Load 从文件加载配置
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()

	ext := filepath.Ext(configPath)
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse yaml config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config format: %s", ext)
	}

	// 环境变量覆盖
	applyEnvOverrides(cfg)

	// 校验配置
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(configPath string, cfg *Config) error {
	// 确保目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// applyEnvOverrides 环境变量覆盖配置
func applyEnvOverrides(cfg *Config) {
	if port := os.Getenv("AUTO_SCAN_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Server.Port)
	}
	if dbPath := os.Getenv("AUTO_SCAN_DB_PATH"); dbPath != "" {
		cfg.Database.Path = dbPath
	}
	if storagePath := os.Getenv("AUTO_SCAN_STORAGE_PATH"); storagePath != "" {
		cfg.Storage.Path = storagePath
	}
	if logLevel := os.Getenv("AUTO_SCAN_LOG_LEVEL"); logLevel != "" {
		cfg.Log.Level = logLevel
	}
}

// Validate 校验配置
func Validate(cfg *Config) error {
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}

	if cfg.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}

	if cfg.Storage.Path == "" {
		return fmt.Errorf("storage path cannot be empty")
	}

	if cfg.Scan.MaxConcurrent < 1 {
		return fmt.Errorf("max concurrent scans must be at least 1")
	}

	if cfg.Device.MonitorInterval < time.Second {
		return fmt.Errorf("monitor interval must be at least 1s")
	}

	validLogLevels := map[string]bool{"debug": true, "info": true, "warning": true, "error": true}
	if !validLogLevels[cfg.Log.Level] {
		return fmt.Errorf("invalid log level: %s", cfg.Log.Level)
	}

	return nil
}