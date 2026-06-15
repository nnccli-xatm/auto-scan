package models

import "time"

// ScanFile 扫描文件模型
type ScanFile struct {
	ID           string    `json:"id" db:"id"`
	TaskID       string    `json:"task_id" db:"task_id"`
	DeviceID     string    `json:"device_id" db:"device_id"`
	Filename     string    `json:"filename" db:"filename"`
	OriginalName string    `json:"original_name" db:"original_name"`
	FilePath     string    `json:"file_path" db:"file_path"`
	FileSize     int64     `json:"file_size" db:"file_size"`
	Checksum     string    `json:"checksum" db:"checksum"` // Blake3
	PageNumber   int       `json:"page_number" db:"page_number"`
	Width        int       `json:"width" db:"width"`
	Height       int       `json:"height" db:"height"`
	Format       string    `json:"format" db:"format"` // JPEG, PDF
	Status       string    `json:"status" db:"status"` // active, archived, deleted
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// FileFormat 文件格式常量
const (
	FileFormatJPEG = "JPEG"
	FileFormatPDF  = "PDF"
)

// FileStatus 文件状态常量
const (
	FileStatusActive   = "active"
	FileStatusArchived = "archived"
	FileStatusDeleted  = "deleted"
)

// FileFilter 文件查询过滤条件
type FileFilter struct {
	DeviceID  string
	TaskID    string
	Format    string
	StartDate *time.Time
	EndDate   *time.Time
	Page      int
	PageSize  int
}

// StorageStats 存储统计
type StorageStats struct {
	TotalFiles  int   `json:"total_files"`
	TotalSize   int64 `json:"total_size"`
	JPEGCount   int   `json:"jpeg_count"`
	PDFCount    int   `json:"pdf_count"`
}
