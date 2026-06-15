-- Auto Scan 数据库 Schema
-- 版本: 1.0.0
-- 数据库: SQLite 3.40+
-- 特性: WAL模式支持，支持并发读写

-- 启用外键约束
PRAGMA foreign_keys = ON;

-- 配置WAL模式以支持并发（应用启动时执行）
-- PRAGMA journal_mode = WAL;
-- PRAGMA synchronous = NORMAL;
-- PRAGMA cache_size = 10000;
-- PRAGMA temp_store = MEMORY;

-- ========================================================
-- 1. 设备表 (devices)
-- ========================================================
CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,                          -- 设备唯一标识 (UUID)
    name TEXT NOT NULL,                           -- 设备显示名称
    ip_address TEXT NOT NULL,                     -- IP地址
    protocol TEXT NOT NULL DEFAULT 'escl',        -- 协议类型: escl, wsd
    model TEXT,                                   -- 设备型号
    vendor TEXT,                                  -- 厂商: HP, Canon, Ricoh, Fujitsu, Brother, Epson
    status TEXT DEFAULT 'offline',                -- 状态: online, offline, busy, error
    capabilities TEXT,                            -- JSON格式: 设备能力信息
    config TEXT,                                  -- JSON格式: 自定义配置
    last_seen DATETIME,                           -- 最后在线时间
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- 约束
    CHECK (protocol IN ('escl', 'wsd')),
    CHECK (status IN ('online', 'offline', 'busy', 'error')),
    CHECK (vendor IN ('HP', 'Canon', 'Ricoh', 'Fujitsu', 'Brother', 'Epson', 'Other'))
);

-- 设备表索引
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_vendor ON devices(vendor);
CREATE INDEX IF NOT EXISTS idx_devices_protocol ON devices(protocol);
CREATE INDEX IF NOT EXISTS idx_devices_ip ON devices(ip_address);

-- ========================================================
-- 2. 扫描任务表 (scan_tasks)
-- ========================================================
CREATE TABLE IF NOT EXISTS scan_tasks (
    id TEXT PRIMARY KEY,                          -- 任务唯一标识 (UUID)
    device_id TEXT NOT NULL,                      -- 关联设备ID
    status TEXT DEFAULT 'pending',                -- 任务状态
    priority INTEGER DEFAULT 5,                   -- 优先级 1-10，数字越小优先级越高
    settings TEXT,                                -- JSON格式: 扫描设置
    result TEXT,                                  -- JSON格式: 扫描结果
    progress INTEGER DEFAULT 0,                   -- 进度百分比 0-100
    total_pages INTEGER DEFAULT 0,                -- 总页数
    scanned_pages INTEGER DEFAULT 0,              -- 已扫描页数
    error_message TEXT,                           -- 错误信息
    started_at DATETIME,                          -- 开始时间
    completed_at DATETIME,                        -- 完成时间
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT DEFAULT 'system',             -- 创建者 (user_id 或 'system')

    -- 外键约束
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,

    -- 约束
    CHECK (status IN ('pending', 'running', 'paused', 'completed', 'failed', 'cancelled')),
    CHECK (priority BETWEEN 1 AND 10),
    CHECK (progress BETWEEN 0 AND 100)
);

-- 任务表索引
CREATE INDEX IF NOT EXISTS idx_tasks_status ON scan_tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_device ON scan_tasks(device_id);
CREATE INDEX IF NOT EXISTS idx_tasks_created ON scan_tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_tasks_status_created ON scan_tasks(status, created_at);

-- ========================================================
-- 3. 扫描文件表 (scan_files)
-- ========================================================
CREATE TABLE IF NOT EXISTS scan_files (
    id TEXT PRIMARY KEY,                          -- 文件唯一标识 (UUID)
    task_id TEXT NOT NULL,                        -- 关联任务ID
    device_id TEXT NOT NULL,                      -- 关联设备ID
    filename TEXT NOT NULL,                       -- 存储文件名
    original_name TEXT,                           -- 原始文件名
    file_path TEXT NOT NULL,                      -- 完整存储路径
    file_size INTEGER,                            -- 文件大小（字节）
    checksum TEXT,                                -- Blake3校验和
    page_number INTEGER,                          -- 页码
    width INTEGER,                                -- 图像宽度
    height INTEGER,                               -- 图像高度
    format TEXT DEFAULT 'JPEG',                   -- 格式: JPEG, PDF
    status TEXT DEFAULT 'active',                 -- 状态: active, archived, deleted
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    -- 外键约束
    FOREIGN KEY (task_id) REFERENCES scan_tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,

    -- 约束
    CHECK (format IN ('JPEG', 'PDF')),
    CHECK (status IN ('active', 'archived', 'deleted')),
    CHECK (file_size >= 0)
);

-- 文件表索引
CREATE INDEX IF NOT EXISTS idx_files_task ON scan_files(task_id);
CREATE INDEX IF NOT EXISTS idx_files_device ON scan_files(device_id);
CREATE INDEX IF NOT EXISTS idx_files_created ON scan_files(created_at);
CREATE INDEX IF NOT EXISTS idx_files_format ON scan_files(format);
CREATE INDEX IF NOT EXISTS idx_files_status ON scan_files(status);

-- ========================================================
-- 4. 审计日志表 (audit_logs)
-- ========================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL,                          -- 日志级别: debug, info, warning, error
    event_type TEXT NOT NULL,                     -- 事件类型
    user_id TEXT,                                 -- 用户ID
    device_id TEXT,                               -- 关联设备ID
    task_id TEXT,                                 -- 关联任务ID
    message TEXT NOT NULL,                        -- 日志消息
    details TEXT,                                 -- JSON格式: 详细信息
    ip_address TEXT,                              -- 客户端IP
    user_agent TEXT,                              -- 客户端User-Agent

    -- 约束
    CHECK (level IN ('debug', 'info', 'warning', 'error'))
);

-- 审计日志表索引
CREATE INDEX IF NOT EXISTS idx_audit_time ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_level ON audit_logs(level);
CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_device ON audit_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_time_level ON audit_logs(timestamp, level);

-- ========================================================
-- 5. 系统配置表 (system_config)
-- ========================================================
CREATE TABLE IF NOT EXISTS system_config (
    key TEXT PRIMARY KEY,                         -- 配置项键
    value TEXT NOT NULL,                          -- 配置项值
    description TEXT,                             -- 配置说明
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认配置
INSERT OR IGNORE INTO system_config (key, value, description) VALUES
('storage.path', '/var/lib/auto-scan/scans', '扫描文件存储路径'),
('storage.max_size', '10737418240', '最大存储空间（字节，默认10GB）'),
('storage.backup_enabled', 'false', '是否启用备份'),
('storage.backup_path', '', '备份存储路径'),
('scan.default_resolution', '300', '默认扫描分辨率（DPI）'),
('scan.default_color_mode', 'Color', '默认色彩模式'),
('scan.max_concurrent', '5', '最大并发扫描数'),
('device.monitor_interval', '2', '设备状态监控间隔（秒）'),
('device.auto_discover', 'true', '是否自动发现设备'),
('log.retention_days', '90', '日志保留天数'),
('api.port', '8080', 'API服务端口'),
('api.cors_origins', '*', 'CORS允许的来源');

-- ========================================================
-- 6. 视图 (Views)
-- ========================================================

-- 设备统计视图
CREATE VIEW IF NOT EXISTS v_device_stats AS
SELECT
    vendor,
    status,
    COUNT(*) as count
FROM devices
GROUP BY vendor, status;

-- 任务统计视图（最近30天）
CREATE VIEW IF NOT EXISTS v_task_stats AS
SELECT
    DATE(created_at) as date,
    status,
    COUNT(*) as count,
    SUM(total_pages) as total_pages
FROM scan_tasks
WHERE created_at >= DATE('now', '-30 days')
GROUP BY DATE(created_at), status;

-- 存储使用统计视图
CREATE VIEW IF NOT EXISTS v_storage_stats AS
SELECT
    COUNT(*) as total_files,
    SUM(file_size) as total_size,
    SUM(CASE WHEN format = 'JPEG' THEN 1 ELSE 0 END) as jpeg_count,
    SUM(CASE WHEN format = 'PDF' THEN 1 ELSE 0 END) as pdf_count
FROM scan_files
WHERE status = 'active';

-- ========================================================
-- 7. 触发器 (Triggers)
-- ========================================================

-- 自动更新 updated_at 字段
CREATE TRIGGER IF NOT EXISTS tr_devices_updated_at
AFTER UPDATE ON devices
BEGIN
    UPDATE devices SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

-- 审计日志自动清理（保留90天）
CREATE TRIGGER IF NOT EXISTS tr_audit_cleanup
AFTER INSERT ON audit_logs
BEGIN
    DELETE FROM audit_logs
    WHERE timestamp < DATETIME('now', '-90 days')
    AND (SELECT COUNT(*) FROM audit_logs) > 100000;  -- 保留最近10万条
END;

-- 任务状态变更记录
CREATE TRIGGER IF NOT EXISTS tr_task_status_change
AFTER UPDATE OF status ON scan_tasks
WHEN OLD.status != NEW.status
BEGIN
    INSERT INTO audit_logs (level, event_type, task_id, device_id, message, details)
    VALUES (
        'info',
        'task.status_changed',
        NEW.id,
        NEW.device_id,
        'Task status changed from ' || OLD.status || ' to ' || NEW.status,
        JSON_OBJECT('old_status', OLD.status, 'new_status', NEW.status)
    );
END;

-- ========================================================
-- 8. 初始化数据
-- ========================================================

-- 创建初始设备（示例数据，生产环境可删除）
-- INSERT OR IGNORE INTO devices (id, name, ip_address, protocol, model, vendor, status)
-- VALUES ('dev_hp_001', 'HP Smart Tank 750', '192.168.3.11', 'escl', 'Smart Tank 750 series', 'HP', 'online');

-- ========================================================
-- 使用说明
-- ========================================================

-- 1. 创建数据库文件
-- sqlite3 auto-scan.db < schema.sql

-- 2. 初始化WAL模式（推荐，应用启动时执行）
-- sqlite3 auto-scan.db "PRAGMA journal_mode=WAL;"

-- 3. 验证表创建
-- .tables
-- .schema devices

-- 4. 常用查询示例

-- 查询在线设备
-- SELECT * FROM devices WHERE status = 'online';

-- 查询最近24小时的任务
-- SELECT * FROM scan_tasks
-- WHERE created_at >= DATETIME('now', '-1 day')
-- ORDER BY created_at DESC;

-- 查询存储使用情况
-- SELECT * FROM v_storage_stats;

-- 查询设备统计
-- SELECT * FROM v_device_stats;

-- 清理已删除文件记录（物理删除文件后执行）
-- DELETE FROM scan_files WHERE status = 'deleted';

-- VACUUM（定期执行，回收空间）
-- VACUUM;

-- ========================================================
-- 版本历史
-- ========================================================
-- v1.0.0 (2026-06-11) - 初始版本
-- - 创建基础表结构
-- - 添加索引和约束
-- - 创建视图和触发器
-- - 添加默认配置
