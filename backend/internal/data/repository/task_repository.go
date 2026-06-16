package repository

import (
	"auto-scan/internal/data/models"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// taskRepository 任务Repository实现
type taskRepository struct {
	db *DB
}

// NewTaskRepository 创建任务Repository
func NewTaskRepository(db *DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(ctx context.Context, task *models.ScanTask) error {
	query := `INSERT INTO scan_tasks (id, device_id, status, priority, settings, result, progress, total_pages, scanned_pages, error_message, started_at, completed_at, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	task.CreatedAt = now
	if task.Status == "" {
		task.Status = "pending"
	}
	_, err := r.db.ExecContext(ctx, query,
		task.ID, task.DeviceID, task.Status, task.Priority,
		task.Settings, task.Result, task.Progress,
		task.TotalPages, task.ScannedPages, task.ErrorMessage,
		task.StartedAt, task.CompletedAt, task.CreatedAt, task.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	return nil
}

func (r *taskRepository) GetByID(ctx context.Context, id string) (*models.ScanTask, error) {
	query := `SELECT id, device_id, status, priority, settings, result, progress, total_pages, scanned_pages, error_message, started_at, completed_at, created_at, created_by
		FROM scan_tasks WHERE id = ?`
	task := &models.ScanTask{}
	var settingsStr, resultStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.DeviceID, &task.Status, &task.Priority,
		&settingsStr, &resultStr, &task.Progress,
		&task.TotalPages, &task.ScannedPages, &task.ErrorMessage,
		&task.StartedAt, &task.CompletedAt, &task.CreatedAt, &task.CreatedBy)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	task.Settings = settingsStr
	task.Result = resultStr
	return task, nil
}

func (r *taskRepository) Update(ctx context.Context, task *models.ScanTask) error {
	query := `UPDATE scan_tasks SET status=?, priority=?, settings=?, result=?, progress=?, total_pages=?, scanned_pages=?, error_message=?, completed_at=? WHERE id=?`
	_, err := r.db.ExecContext(ctx, query,
		task.Status, task.Priority, task.Settings, task.Result,
		task.Progress, task.TotalPages, task.ScannedPages,
		task.ErrorMessage, task.CompletedAt, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}

func (r *taskRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM scan_tasks WHERE id=?", id)
	return err
}

func (r *taskRepository) List(ctx context.Context, filter TaskFilter) ([]*models.ScanTask, int, error) {
	where := "1=1"
	args := []interface{}{}
	if filter.Status != "" {
		where += " AND status=?"
		args = append(args, filter.Status)
	}
	if filter.DeviceID != "" {
		where += " AND device_id=?"
		args = append(args, filter.DeviceID)
	}

	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scan_tasks WHERE "+where, args...).Scan(&total)

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := `SELECT id, device_id, status, priority, settings, result, progress, total_pages, scanned_pages, error_message, started_at, completed_at, created_at, created_by
		FROM scan_tasks WHERE ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := []*models.ScanTask{}
	for rows.Next() {
		t := &models.ScanTask{}
		var s, r string
		if err := rows.Scan(&t.ID, &t.DeviceID, &t.Status, &t.Priority,
			&s, &r, &t.Progress, &t.TotalPages, &t.ScannedPages, &t.ErrorMessage,
			&t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.CreatedBy); err != nil {
			return nil, 0, err
		}
		t.Settings = s
		t.Result = r
		tasks = append(tasks, t)
	}
	return tasks, total, nil
}

func (r *taskRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE scan_tasks SET status=?, completed_at=? WHERE id=?",
		status, time.Now(), id)
	return err
}

func (r *taskRepository) UpdateProgress(ctx context.Context, id string, progress, scannedPages, totalPages int) error {
	_, err := r.db.ExecContext(ctx, "UPDATE scan_tasks SET progress=?, scanned_pages=?, total_pages=? WHERE id=?",
		progress, scannedPages, totalPages, id)
	return err
}