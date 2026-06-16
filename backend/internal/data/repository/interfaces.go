package repository

import (
	"auto-scan/internal/data/models"
	"context"
)

// TaskRepository 任务数据访问接口
type TaskRepository interface {
	Create(ctx context.Context, task *models.ScanTask) error
	GetByID(ctx context.Context, id string) (*models.ScanTask, error)
	Update(ctx context.Context, task *models.ScanTask) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter TaskFilter) ([]*models.ScanTask, int, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateProgress(ctx context.Context, id string, progress, scannedPages, totalPages int) error
}

// TaskFilter 任务查询过滤
type TaskFilter struct {
	Status   string
	DeviceID string
	Page     int
	PageSize int
}

// FileRepository 文件数据访问接口
type FileRepository interface {
	Create(ctx context.Context, file *models.ScanFile) error
	GetByID(ctx context.Context, id string) (*models.ScanFile, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, filter FileFilter) ([]*models.ScanFile, int, error)
	BatchDelete(ctx context.Context, ids []string) error
}

// FileFilter 文件查询过滤
type FileFilter struct {
	DeviceID  string
	TaskID    string
	Format    string
	StartDate string
	EndDate   string
	Page      int
	PageSize  int
}
