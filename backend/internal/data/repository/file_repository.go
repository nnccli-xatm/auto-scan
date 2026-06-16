package repository

import (
	"auto-scan/internal/data/models"
	"context"
	"database/sql"
	"fmt"
	"time"
)

// fileRepository 文件Repository实现
type fileRepository struct {
	db *DB
}

// NewFileRepository 创建文件Repository
func NewFileRepository(db *DB) FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) Create(ctx context.Context, file *models.ScanFile) error {
	query := `INSERT INTO scan_files (id, task_id, device_id, filename, original_name, file_path, file_size, checksum, page_number, width, height, format, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	now := time.Now()
	file.CreatedAt = now
	_, err := r.db.ExecContext(ctx, query,
		file.ID, file.TaskID, file.DeviceID, file.Filename,
		file.OriginalName, file.FilePath, file.FileSize, file.Checksum,
		file.PageNumber, file.Width, file.Height, file.Format, file.Status, file.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	return nil
}

func (r *fileRepository) GetByID(ctx context.Context, id string) (*models.ScanFile, error) {
	query := `SELECT id, task_id, device_id, filename, file_path, file_size, checksum, page_number, width, height, format, status, created_at
		FROM scan_files WHERE id = ?`
	f := &models.ScanFile{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&f.ID, &f.TaskID, &f.DeviceID, &f.Filename,
		&f.FilePath, &f.FileSize, &f.Checksum,
		&f.PageNumber, &f.Width, &f.Height, &f.Format, &f.Status, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}
	return f, nil
}

func (r *fileRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM scan_files WHERE id=?", id)
	return err
}

func (r *fileRepository) List(ctx context.Context, filter FileFilter) ([]*models.ScanFile, int, error) {
	where := "1=1"
	args := []interface{}{}
	if filter.DeviceID != "" {
		where += " AND device_id=?"
		args = append(args, filter.DeviceID)
	}
	if filter.TaskID != "" {
		where += " AND task_id=?"
		args = append(args, filter.TaskID)
	}

	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM scan_files WHERE "+where, args...).Scan(&total)

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := `SELECT id, task_id, device_id, filename, file_path, file_size, checksum, page_number, width, height, format, status, created_at
		FROM scan_files WHERE ` + where + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	files := []*models.ScanFile{}
	for rows.Next() {
		f := &models.ScanFile{}
		f.OriginalName = ""
		if err := rows.Scan(&f.ID, &f.TaskID, &f.DeviceID, &f.Filename,
			&f.FilePath, &f.FileSize, &f.Checksum, &f.PageNumber,
			&f.Width, &f.Height, &f.Format, &f.Status, &f.CreatedAt); err != nil {
			return nil, 0, err
		}
		files = append(files, f)
	}
	return files, total, nil
}

func (r *fileRepository) BatchDelete(ctx context.Context, ids []string) error {
	return r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, "DELETE FROM scan_files WHERE id=?")
		if err != nil {
			return err
		}
		defer stmt.Close()
		for _, id := range ids {
			stmt.ExecContext(ctx, id)
		}
		return nil
	})
}
