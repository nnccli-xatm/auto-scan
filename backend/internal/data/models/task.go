package models

import "time"

// ScanTask 扫描任务模型
type ScanTask struct {
	ID             string    `json:"id" db:"id"`
	DeviceID       string    `json:"device_id" db:"device_id"`
	Status         string    `json:"status" db:"status"` // pending, running, paused, completed, failed, cancelled
	Priority       int       `json:"priority" db:"priority"` // 1-10
	Settings       string    `json:"settings" db:"settings"`       // JSON: 扫描设置
	Result         string    `json:"result" db:"result"`             // JSON: 扫描结果
	Progress       int       `json:"progress" db:"progress"`         // 0-100
	TotalPages     int       `json:"total_pages" db:"total_pages"`
	ScannedPages   int       `json:"scanned_pages" db:"scanned_pages"`
	ErrorMessage   string    `json:"error_message" db:"error_message"`
	StartedAt      time.Time `json:"started_at" db:"started_at"`
	CompletedAt    time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	CreatedBy      string    `json:"created_by" db:"created_by"`
}

// TaskStatus 任务状态常量
const (
	TaskStatusPending    = "pending"
	TaskStatusRunning    = "running"
	TaskStatusPaused     = "paused"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusCancelled  = "cancelled"
)

// ScanSettings 扫描设置结构
type ScanSettings struct {
	Resolution   int    `json:"resolution"`    // DPI: 75, 100, 150, 200, 300, 400, 600
	ColorMode    string `json:"color_mode"`    // Color, Grayscale, BW
	Format       string `json:"format"`        // JPEG, PDF
	InputSource  string `json:"input_source"`  // Platen, Feeder
}

// ScanResult 扫描结果结构
type ScanResult struct {
	Files       []string `json:"files"`        // 文件ID列表
	StoragePath string   `json:"storage_path"` // 存储路径
	TotalPages  int      `json:"total_pages"`
	FileSize    int64    `json:"file_size"`    // 总文件大小
}

// DefaultScanSettings 默认扫描设置
var DefaultScanSettings = ScanSettings{
	Resolution:  300,
	ColorMode:   "Color",
	Format:      "JPEG",
	InputSource: "Feeder",
}
