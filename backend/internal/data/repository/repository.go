// Repository层 - 数据访问对象
// 设计原则：
// 1. 面向接口编程，便于测试和替换
// 2. 每个模型对应一个Repository
// 3. 支持事务操作
// 4. 返回具体错误类型，便于上层处理

package repository

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB 数据库连接
type DB struct {
	*sql.DB
}

// Initialize 初始化数据库
func Initialize(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(1) // SQLite单连接
	db.SetMaxIdleConns(1)

	// 启用外键
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &DB{db}, nil
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.DB.Close()
}

// WithTransaction 执行事务
func (db *DB) WithTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Common errors
var (
	ErrNotFound = fmt.Errorf("record not found")
	ErrExists   = fmt.Errorf("record already exists")
	ErrDatabase = fmt.Errorf("database error")
)