package repository

import (
	"auto-scan/internal/data/models"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// DeviceRepository 设备数据访问接口
type DeviceRepository interface {
	// 基础CRUD
	Create(ctx context.Context, device *models.Device) error
	GetByID(ctx context.Context, id string) (*models.Device, error)
	GetByIPAddress(ctx context.Context, ip string) (*models.Device, error)
	Update(ctx context.Context, device *models.Device) error
	Delete(ctx context.Context, id string) error

	// 列表查询
	List(ctx context.Context, filter DeviceFilter) ([]*models.Device, int, error)
	ListByStatus(ctx context.Context, status string) ([]*models.Device, error)

	// 状态管理
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateLastSeen(ctx context.Context, id string) error

	// 批量操作
	BatchCreate(ctx context.Context, devices []*models.Device) error
	BatchDelete(ctx context.Context, ids []string) error
}

// DeviceFilter 设备查询过滤条件
type DeviceFilter struct {
	Status   string
	Vendor   string
	Protocol string
	Page     int
	PageSize int
}

// deviceRepository 设备Repository实现
type deviceRepository struct {
	db *DB
}

// NewDeviceRepository 创建设备Repository
func NewDeviceRepository(db *DB) DeviceRepository {
	return &deviceRepository{db: db}
}

// Create 创建设备
func (r *deviceRepository) Create(ctx context.Context, device *models.Device) error {
	query := `
		INSERT INTO devices (id, name, ip_address, protocol, model, vendor, status, capabilities, config, last_seen, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	if device.Status == "" {
		device.Status = models.DeviceStatusOffline
	}

	capabilitiesJSON, _ := json.Marshal(device.Capabilities)
	configJSON, _ := json.Marshal(device.Config)

	_, err := r.db.ExecContext(ctx, query,
		device.ID, device.Name, device.IPAddress, device.Protocol,
		device.Model, device.Vendor, device.Status,
		string(capabilitiesJSON), string(configJSON),
		device.LastSeen, device.CreatedAt, device.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create device: %w", err)
	}

	return nil
}

// GetByID 根据ID获取设备
func (r *deviceRepository) GetByID(ctx context.Context, id string) (*models.Device, error) {
	query := `
		SELECT id, name, ip_address, protocol, model, vendor, status, capabilities, config, last_seen, created_at, updated_at
		FROM devices
		WHERE id = ?
	`

	device := &models.Device{}
	var capabilitiesJSON, configJSON string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&device.ID, &device.Name, &device.IPAddress, &device.Protocol,
		&device.Model, &device.Vendor, &device.Status,
		&capabilitiesJSON, &configJSON,
		&device.LastSeen, &device.CreatedAt, &device.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	device.Capabilities = capabilitiesJSON
	device.Config = configJSON

	return device, nil
}

// GetByIPAddress 根据IP地址获取设备
func (r *deviceRepository) GetByIPAddress(ctx context.Context, ip string) (*models.Device, error) {
	query := `
		SELECT id, name, ip_address, protocol, model, vendor, status, capabilities, config, last_seen, created_at, updated_at
		FROM devices
		WHERE ip_address = ?
	`

	device := &models.Device{}
	var capabilitiesJSON, configJSON string

	err := r.db.QueryRowContext(ctx, query, ip).Scan(
		&device.ID, &device.Name, &device.IPAddress, &device.Protocol,
		&device.Model, &device.Vendor, &device.Status,
		&capabilitiesJSON, &configJSON,
		&device.LastSeen, &device.CreatedAt, &device.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}

	device.Capabilities = capabilitiesJSON
	device.Config = configJSON

	return device, nil
}

// Update 更新设备
func (r *deviceRepository) Update(ctx context.Context, device *models.Device) error {
	query := `
		UPDATE devices
		SET name = ?, ip_address = ?, protocol = ?, model = ?, vendor = ?,
		    status = ?, capabilities = ?, config = ?, last_seen = ?, updated_at = ?
		WHERE id = ?
	`

	device.UpdatedAt = time.Now()

	capabilitiesJSON, _ := json.Marshal(device.Capabilities)
	configJSON, _ := json.Marshal(device.Config)

	result, err := r.db.ExecContext(ctx, query,
		device.Name, device.IPAddress, device.Protocol,
		device.Model, device.Vendor, device.Status,
		string(capabilitiesJSON), string(configJSON),
		device.LastSeen, device.UpdatedAt, device.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// Delete 删除设备
func (r *deviceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM devices WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// List 分页查询设备列表
func (r *deviceRepository) List(ctx context.Context, filter DeviceFilter) ([]*models.Device, int, error) {
	whereClause := "1=1"
	args := []interface{}{}

	if filter.Status != "" {
		whereClause += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Vendor != "" {
		whereClause += " AND vendor = ?"
		args = append(args, filter.Vendor)
	}
	if filter.Protocol != "" {
		whereClause += " AND protocol = ?"
		args = append(args, filter.Protocol)
	}

	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM devices WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count devices: %w", err)
	}

	// 查询数据
	query := fmt.Sprintf(`
		SELECT id, name, ip_address, protocol, model, vendor, status, capabilities, config, last_seen, created_at, updated_at
		FROM devices
		WHERE %s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	devices := []*models.Device{}
	for rows.Next() {
		device := &models.Device{}
		var capabilitiesJSON, configJSON string

		if err := rows.Scan(
			&device.ID, &device.Name, &device.IPAddress, &device.Protocol,
			&device.Model, &device.Vendor, &device.Status,
			&capabilitiesJSON, &configJSON,
			&device.LastSeen, &device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan device: %w", err)
		}

		device.Capabilities = capabilitiesJSON
		device.Config = configJSON
		devices = append(devices, device)
	}

	return devices, total, nil
}

// ListByStatus 根据状态查询设备
func (r *deviceRepository) ListByStatus(ctx context.Context, status string) ([]*models.Device, error) {
	query := `
		SELECT id, name, ip_address, protocol, model, vendor, status, capabilities, config, last_seen, created_at, updated_at
		FROM devices
		WHERE status = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	devices := []*models.Device{}
	for rows.Next() {
		device := &models.Device{}
		var capabilitiesJSON, configJSON string

		if err := rows.Scan(
			&device.ID, &device.Name, &device.IPAddress, &device.Protocol,
			&device.Model, &device.Vendor, &device.Status,
			&capabilitiesJSON, &configJSON,
			&device.LastSeen, &device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}

		device.Capabilities = capabilitiesJSON
		device.Config = configJSON
		devices = append(devices, device)
	}

	return devices, nil
}

// UpdateStatus 更新设备状态
func (r *deviceRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE devices SET status = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateLastSeen 更新最后在线时间
func (r *deviceRepository) UpdateLastSeen(ctx context.Context, id string) error {
	query := `UPDATE devices SET last_seen = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, time.Now(), time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update device last seen: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// BatchCreate 批量创建设备
func (r *deviceRepository) BatchCreate(ctx context.Context, devices []*models.Device) error {
	return r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO devices (id, name, ip_address, protocol, model, vendor, status, capabilities, config, last_seen, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, device := range devices {
			now := time.Now()
			device.CreatedAt = now
			device.UpdatedAt = now

			capabilitiesJSON, _ := json.Marshal(device.Capabilities)
			configJSON, _ := json.Marshal(device.Config)

			if _, err := stmt.ExecContext(ctx,
				device.ID, device.Name, device.IPAddress, device.Protocol,
				device.Model, device.Vendor, device.Status,
				string(capabilitiesJSON), string(configJSON),
				device.LastSeen, device.CreatedAt, device.UpdatedAt,
			); err != nil {
				return fmt.Errorf("failed to create device %s: %w", device.ID, err)
			}
		}

		return nil
	})
}

// BatchDelete 批量删除设备
func (r *deviceRepository) BatchDelete(ctx context.Context, ids []string) error {
	return r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `DELETE FROM devices WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()

		for _, id := range ids {
			if _, err := stmt.ExecContext(ctx, id); err != nil {
				return fmt.Errorf("failed to delete device %s: %w", id, err)
			}
		}

		return nil
	})
}
