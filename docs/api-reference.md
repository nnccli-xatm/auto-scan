# Auto Scan 后端服务接口文档

> **版本**: v1.0.0  
> **更新日期**: 2026-06-11  
> **基础URL**: `http://<host>:8080/api/v1`  
> **适用对象**: 外部调用程序、第三方集成、前端开发者

本文档定义了 Auto Scan 扫描设备自动任务系统的所有 HTTP 接口规范，供外部程序开发对接使用。

---

## 目录

1. [快速开始](#1-快速开始)
2. [通用规范](#2-通用规范)
3. [统一响应格式](#3-统一响应格式)
4. [认证机制](#4-认证机制)
5. [设备管理接口](#5-设备管理接口)
6. [任务管理接口](#6-任务管理接口)
7. [文件管理接口](#7-文件管理接口)
8. [系统管理接口](#8-系统管理接口)
9. [实时通知（WebSocket）](#9-实时通知websocket)
10. [数据模型](#10-数据模型)
11. [错误码参考](#11-错误码参考)
12. [调用示例](#12-调用示例)

---

## 1. 快速开始

### 最简调用流程

```
1. 发现/添加设备    →  POST /devices/discover  或  POST /devices
2. 创建扫描任务    →  POST /tasks
3. 轮询任务进度    →  GET  /tasks/{id}/progress
4. 下载扫描文件    →  GET  /files/{id}/download
```

### Hello World 示例

```bash
# 获取系统状态（无需认证）
curl http://localhost:8080/api/v1/system/status

# 添加一台扫描仪
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "HP Smart Tank 750",
    "ip_address": "192.168.3.11",
    "protocol": "escl"
  }'
```

---

## 2. 通用规范

### 2.1 基础信息

| 项目 | 说明 |
|------|------|
| **协议** | HTTP/1.1 |
| **基础路径** | `/api/v1` |
| **数据格式** | JSON（`Content-Type: application/json`） |
| **字符编码** | UTF-8 |
| **字段命名** | `snake_case`（如 `ip_address`、`created_at`） |
| **时间格式** | RFC 3339（如 `2026-06-11T14:30:25Z`） |
| **ID 格式** | UUID v4 |
| **分页参数** | `page`（页码，从1开始）、`page_size`（每页条数，最大100） |

### 2.2 请求头

| 请求头 | 必填 | 说明 |
|--------|------|------|
| `Content-Type` | 是（写操作） | 固定 `application/json` |
| `Authorization` | 是（需认证接口） | `Bearer <token>` |
| `X-Request-ID` | 否 | 请求追踪ID，响应中原样返回 |

### 2.3 HTTP 方法语义

| 方法 | 语义 | 幂等性 |
|------|------|--------|
| `GET` | 查询资源 | 是 |
| `POST` | 创建资源/触发动作 | 否 |
| `PUT` | 更新资源（整体） | 是 |
| `DELETE` | 删除资源 | 是 |

### 2.4 限流

- 默认限流：每 IP 每分钟 60 次请求
- 超出限流返回 `429 Too Many Requests`，错误码 `429001`
- 生产环境建议对轮询类接口添加客户端缓存

---

## 3. 统一响应格式

### 3.1 成功响应

所有接口返回统一的 JSON 结构：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `code` | int | 业务状态码，`0` 表示成功 |
| `message` | string | 状态描述 |
| `data` | object/array | 响应数据，失败时可能为 `null` |

### 3.2 分页响应

列表查询接口返回分页结构：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [ ... ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

### 3.3 错误响应

```json
{
  "code": 1001001,
  "message": "device not found",
  "data": {
    "details": "device with id xxx does not exist"
  }
}
```

> HTTP 状态码与业务错误码配合使用。HTTP 状态码反映请求层面状态，业务 `code` 反映业务语义。完整错误码见 [第11节](#11-错误码参考)。

---

## 4. 认证机制

### 4.1 JWT Token

系统使用 JWT（JSON Web Token）进行认证。

**获取 Token**（如系统启用认证）：

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "xxx"}'
```

**使用 Token**：

```bash
curl -H "Authorization: Bearer <your_token>" \
  http://localhost:8080/api/v1/devices
```

### 4.2 Token 规范

| 属性 | 值 |
|------|-----|
| 签名算法 | HS256 |
| 有效期 | 7 天 |
| 携带字段 | `user_id`、`username`、`role` |
| 传递方式 | `Authorization: Bearer <token>` |

### 4.3 认证失败

| HTTP 状态码 | 业务码 | 说明 |
|-------------|--------|------|
| 401 | `401001` | 缺少 Authorization 头 |
| 401 | `401002` | Authorization 格式错误 |
| 401 | `401003` | Token 无效或已过期 |

> **说明**：当前 MVP 阶段设备/任务/文件/系统接口可匿名访问，生产部署建议启用认证中间件。

---

## 5. 设备管理接口

设备是扫描任务的执行主体。支持自动发现和手动添加。

### 5.1 获取设备列表

```
GET /devices
```

**查询参数**：

| 参数 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `page` | int | 否 | 1 | 页码 |
| `page_size` | int | 否 | 20 | 每页数量（最大100） |
| `status` | string | 否 | - | 按状态过滤：`online`/`offline`/`busy`/`error` |
| `vendor` | string | 否 | - | 按厂商过滤：`HP`/`Canon`/`Ricoh`/`Fujitsu`/`Brother`/`Epson` |
| `protocol` | string | 否 | - | 按协议过滤：`escl`/`wsd` |

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "HP Smart Tank 750",
        "ip_address": "192.168.3.11",
        "protocol": "escl",
        "model": "Smart Tank 750 series",
        "vendor": "HP",
        "status": "online",
        "capabilities": "{...}",
        "config": "{}",
        "last_seen": "2026-06-11T14:30:25Z",
        "created_at": "2026-06-11T10:00:00Z",
        "updated_at": "2026-06-11T14:30:25Z"
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total": 1,
      "total_pages": 1
    }
  }
}
```

---

### 5.2 发现设备

通过 mDNS 扫描局域网，自动发现支持 eSCL 协议的扫描仪。

```
POST /devices/discover
```

**请求体**：无

**响应**：

```json
{
  "code": 0,
  "message": "discovery started",
  "data": {
    "found": 2,
    "devices": [
      {
        "id": "...",
        "name": "HP Smart Tank 750 series",
        "ip_address": "192.168.3.11",
        "protocol": "escl",
        "vendor": "HP",
        "model": "Smart Tank 750 series"
      }
    ]
  }
}
```

> **说明**：发现过程约需 5 秒，接口会阻塞直到发现完成。发现的设备会自动入库。

---

### 5.3 添加设备

手动添加一台指定 IP 的扫描设备。

```
POST /devices
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 设备名称（最长100字符） |
| `ip_address` | string | 是 | IPv4 地址 |
| `protocol` | string | 是 | 协议：`escl` 或 `wsd` |

**示例**：

```json
{
  "name": "办公室佳能扫描仪",
  "ip_address": "192.168.3.20",
  "protocol": "escl"
}
```

**响应**：返回创建的 [Device](#101-device-设备) 对象，HTTP 201。

**错误码**：`1001005`（设备已存在）、`1001006`（连接失败）

---

### 5.4 获取设备详情

```
GET /devices/{id}
```

**路径参数**：

| 参数 | 说明 |
|------|------|
| `id` | 设备 ID |

**响应**：返回 [Device](#101-device-设备) 对象。

**错误码**：`1001001`（设备不存在）

---

### 5.5 更新设备

```
PUT /devices/{id}
```

**请求体**（所有字段可选）：

```json
{
  "name": "新名称",
  "config": { "key": "value" }
}
```

---

### 5.6 删除设备

```
DELETE /devices/{id}
```

**响应**：HTTP 204（无内容）或包含删除确认的 JSON。

> **说明**：删除设备会同时清理该设备关联的监控协程。关联的扫描文件记录会因外键级联被删除（物理文件保留）。

---

### 5.7 获取设备实时状态

```
GET /devices/{id}/status
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "device_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "online",
    "adf_status": "loaded",
    "scanner_state": "idle",
    "current_task": "",
    "last_seen": "2026-06-11T14:30:25Z"
  }
}
```

**字段说明**：

| 字段 | 可选值 | 说明 |
|------|--------|------|
| `status` | `online`/`offline`/`busy`/`error` | 设备整体状态 |
| `adf_status` | `empty`/`loaded`/`scanning` | 自动输稿器纸张状态（仅支持ADF的设备有效） |
| `scanner_state` | `idle`/`processing` | 扫描仪工作状态 |
| `current_task` | UUID 或空 | 当前正在执行的任务ID |

> **说明**：对于无ADF设备（如HP DeskJet 4530），`adf_status` 始终为 `empty`，应忽略该字段。
> 设备信息中的 `capabilities.supports_adf` 字段标识设备是否支持ADF。

---

### 5.8 连接设备

主动建立与设备的连接，并刷新其能力信息。

```
POST /devices/{id}/connect
```

**错误码**：`1001001`（不存在）、`1001006`（连接失败）、`503001`（设备不可达）

---

### 5.9 断开设备

```
POST /devices/{id}/disconnect
```

---

## 6. 任务管理接口

任务是扫描的执行单元。创建任务后由调度器异步执行。

### 6.1 创建扫描任务

```
POST /tasks
```

**请求体**：

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `device_id` | string | 是 | - | 目标设备ID |
| `priority` | int | 否 | 5 | 优先级（1-10，数字越小优先级越高） |
| `settings` | object | 否 | 默认设置 | 扫描参数，见 [ScanSettings](#103-scansettings-扫描设置) |

**示例**：

```json
{
  "device_id": "550e8400-e29b-41d4-a716-446655440000",
  "priority": 3,
  "settings": {
    "resolution": 300,
    "color_mode": "Color",
    "format": "JPEG",
    "input_source": "Feeder"
  }
}
```

**响应**：返回创建的 [ScanTask](#102-scantask-扫描任务) 对象，HTTP 201。

**错误码**：
- `1001001`：设备不存在
- `1001002`：设备离线
- `1001003`：设备忙碌
- `1002005`：任务队列已满

> **执行模型**：任务创建后进入队列，调度器按优先级取出执行。ADF 检测到纸张后自动开始扫描。

---

### 6.2 获取任务列表

```
GET /tasks
```

**查询参数**：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | 否 | 页码 |
| `page_size` | int | 否 | 每页数量 |
| `status` | string | 否 | 按状态过滤 |
| `device_id` | string | 否 | 按设备过滤 |

**响应**：返回分页的 [ScanTask](#102-scantask-扫描任务) 列表。

---

### 6.3 获取任务详情

```
GET /tasks/{id}
```

**响应**：返回 [ScanTask](#102-scantask-扫描任务) 对象。

---

### 6.4 获取任务进度

专为轮询设计的轻量接口，返回实时进度。

```
GET /tasks/{id}/progress
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "...",
    "status": "running",
    "progress": 60,
    "scanned_pages": 3,
    "total_pages": 5,
    "current_file": "page_003.jpg",
    "started_at": "2026-06-11T14:30:00Z",
    "estimated_end": "2026-06-11T14:31:30Z"
  }
}
```

> **建议**：轮询间隔 1-2 秒。或使用 [WebSocket](#9-实时通知websocket) 接收实时推送，避免轮询。

---

### 6.5 取消任务

```
DELETE /tasks/{id}
```

**说明**：仅 `pending` 或 `running` 状态的任务可取消。

**错误码**：`1002003`（当前状态不可取消）

---

## 7. 文件管理接口

管理扫描生成的文件，支持预览、下载、批量操作。

### 7.1 获取文件列表

```
GET /files
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码 |
| `page_size` | int | 每页数量 |
| `device_id` | string | 按设备过滤 |
| `task_id` | string | 按任务过滤 |
| `format` | string | 按格式过滤：`JPEG`/`PDF` |
| `start_date` | date | 起始日期（`2026-06-01`） |
| `end_date` | date | 结束日期 |

**响应**：返回分页的 [ScanFile](#104-scanfile-扫描文件) 列表。

---

### 7.2 获取文件详情

```
GET /files/{id}
```

**响应**：返回 [ScanFile](#104-scanfile-扫描文件) 对象。

---

### 7.3 下载文件

```
GET /files/{id}/download
```

**支持断点续传**：可通过 `Range: bytes=0-1023` 请求头请求部分内容，返回 `206 Partial Content`。

**响应**：文件二进制流，`Content-Type` 为 `image/jpeg` 或 `application/pdf`。

**示例**：

```bash
# 直接下载
curl -o scan.jpg http://localhost:8080/api/v1/files/{id}/download

# 断点续传
curl -H "Range: bytes=0-1023" \
  -o part.bin \
  http://localhost:8080/api/v1/files/{id}/download
```

---

### 7.4 批量下载

将多个文件打包为 ZIP 下载。

```
POST /files/batch/download
```

**请求体**：

```json
{
  "file_ids": ["id1", "id2", "id3"]
}
```

**响应**：`Content-Type: application/zip` 的二进制流。

---

### 7.5 删除文件

```
DELETE /files/{id}
```

**响应**：HTTP 204。

---

### 7.6 批量删除

```
POST /files/batch/delete
```

**请求体**：

```json
{
  "file_ids": ["id1", "id2"]
}
```

---

## 8. 系统管理接口

### 8.1 获取系统状态

```
GET /system/status
```

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "version": "1.0.0",
    "uptime": 86400,
    "go_version": "go1.22.0",
    "platform": "darwin/arm64",
    "devices": {
      "total": 5,
      "online": 3,
      "offline": 1,
      "busy": 1,
      "error": 0
    },
    "tasks": {
      "pending": 2,
      "running": 1,
      "completed": 150,
      "failed": 3,
      "cancelled": 5
    },
    "storage": {
      "total": 10737418240,
      "used": 2147483648,
      "free": 8589934592,
      "file_count": 1200
    }
  }
}
```

---

### 8.2 获取审计日志

```
GET /system/logs
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `page` | int | 页码 |
| `page_size` | int | 每页数量 |
| `level` | string | 日志级别：`debug`/`info`/`warning`/`error` |
| `start_date` | datetime | 起始时间 |
| `end_date` | datetime | 结束时间 |

**响应**：返回分页的 [AuditLog](#105-auditlog-审计日志) 列表。

---

### 8.3 获取系统配置

```
GET /system/config
```

---

### 8.4 更新系统配置

```
PUT /system/config
```

**请求体**：完整的配置对象（结构同 GET 返回）。

> **说明**：配置更新支持热重载，无需重启服务。

---

## 9. 实时通知（WebSocket）

对于需要实时感知设备状态变化和任务进度的场景，推荐使用 WebSocket 替代轮询。

### 9.1 建立连接

```
WS /ws
```

### 9.2 订阅事件

连接后发送订阅消息：

```json
{
  "action": "subscribe",
  "channels": ["device.events", "task.progress"]
}
```

### 9.3 事件类型

| 事件 | 触发时机 |
|------|---------|
| `device.status_changed` | 设备状态变化（上线/离线/忙碌） |
| `device.adf_changed` | ADF 纸张状态变化 |
| `task.created` | 任务创建 |
| `task.started` | 任务开始执行 |
| `task.progressed` | 任务进度更新 |
| `task.completed` | 任务完成 |
| `task.failed` | 任务失败 |

**事件消息格式**：

```json
{
  "type": "task.progressed",
  "timestamp": "2026-06-11T14:30:25Z",
  "data": {
    "task_id": "...",
    "device_id": "...",
    "progress": 60,
    "scanned_pages": 3
  }
}
```

---

## 10. 数据模型

### 10.1 Device（设备）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 设备唯一标识 |
| `name` | string | 设备名称 |
| `ip_address` | string | IP 地址 |
| `protocol` | string | 协议：`escl`/`wsd` |
| `model` | string | 设备型号 |
| `vendor` | string | 厂商 |
| `status` | string | 状态：`online`/`offline`/`busy`/`error` |
| `capabilities` | string (JSON) | 设备能力信息 |
| `config` | string (JSON) | 自定义配置 |
| `last_seen` | datetime | 最后在线时间 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

---

### 10.2 ScanTask（扫描任务）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 任务唯一标识 |
| `device_id` | string (UUID) | 关联设备ID |
| `status` | string | 状态：`pending`/`running`/`paused`/`completed`/`failed`/`cancelled` |
| `priority` | int | 优先级（1-10） |
| `settings` | string (JSON) | 扫描设置 |
| `result` | string (JSON) | 扫描结果 |
| `progress` | int | 进度（0-100） |
| `total_pages` | int | 总页数 |
| `scanned_pages` | int | 已扫描页数 |
| `error_message` | string | 错误信息 |
| `started_at` | datetime | 开始时间 |
| `completed_at` | datetime | 完成时间 |
| `created_at` | datetime | 创建时间 |
| `created_by` | string | 创建者（`system` 或用户ID） |

---

### 10.3 ScanSettings（扫描设置）

| 字段 | 类型 | 可选值 | 默认 |
|------|------|--------|------|
| `resolution` | int | `75`/`100`/`150`/`200`/`300`/`400`/`600` | `300` |
| `color_mode` | string | `Color`/`Grayscale`/`BW` | `Color` |
| `format` | string | `JPEG`/`PDF` | `JPEG` |
| `input_source` | string | `Platen`（平板）/`Feeder`（输稿器） | `Feeder` |

---

### 10.4 ScanFile（扫描文件）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string (UUID) | 文件唯一标识 |
| `task_id` | string (UUID) | 关联任务ID |
| `device_id` | string (UUID) | 关联设备ID |
| `filename` | string | 存储文件名 |
| `original_name` | string | 原始文件名 |
| `file_path` | string | 存储路径 |
| `file_size` | int | 文件大小（字节） |
| `checksum` | string | Blake3 校验和 |
| `page_number` | int | 页码 |
| `width` | int | 图像宽度 |
| `height` | int | 图像高度 |
| `format` | string | 格式：`JPEG`/`PDF` |
| `status` | string | 状态：`active`/`archived`/`deleted` |
| `created_at` | datetime | 创建时间 |

---

### 10.5 AuditLog（审计日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 日志ID |
| `timestamp` | datetime | 时间戳 |
| `level` | string | 级别：`debug`/`info`/`warning`/`error` |
| `event_type` | string | 事件类型（见下表） |
| `user_id` | string | 用户ID |
| `device_id` | string | 关联设备ID |
| `task_id` | string | 关联任务ID |
| `message` | string | 日志消息 |
| `details` | object | 详细信息 |
| `ip_address` | string | 客户端IP |

**常见事件类型**：`device.created`、`device.updated`、`device.deleted`、`device.connected`、`task.created`、`task.started`、`task.completed`、`task.failed`、`task.cancelled`、`file.downloaded`、`file.deleted`、`config.updated`

---

## 11. 错误码参考

### 11.1 通用错误码

| 业务码 | HTTP | 说明 |
|--------|------|------|
| `0` | 200 | 成功 |
| `1` | 500 | 未知错误 |
| `400001` | 400 | 请求参数错误 |
| `401001` | 401 | 未授权（缺少Token） |
| `403001` | 403 | 禁止访问 |
| `404001` | 404 | 资源不存在 |
| `405001` | 405 | 方法不允许 |
| `409001` | 409 | 资源冲突 |
| `422001` | 422 | 参数校验失败 |
| `429001` | 429 | 请求过于频繁 |
| `500001` | 500 | 服务器内部错误 |
| `503001` | 503 | 服务不可用 |

### 11.2 设备错误码（1001xxx）

| 业务码 | 说明 |
|--------|------|
| `1001001` | 设备不存在 |
| `1001002` | 设备离线 |
| `1001003` | 设备忙碌 |
| `1001004` | 设备错误 |
| `1001005` | 设备已存在 |
| `1001006` | 设备连接失败 |
| `1001007` | 设备不支持 |

### 11.3 任务错误码（1002xxx）

| 业务码 | 说明 |
|--------|------|
| `1002001` | 任务不存在 |
| `1002002` | 任务创建失败 |
| `1002003` | 任务取消失败（当前状态不可取消） |
| `1002004` | 任务已在运行 |
| `1002005` | 任务队列已满 |

### 11.4 文件错误码（1003xxx）

| 业务码 | 说明 |
|--------|------|
| `1003001` | 文件不存在 |
| `1003002` | 文件下载失败 |
| `1003003` | 文件删除失败 |
| `1003004` | 文件上传失败 |
| `1003005` | 存储空间已满 |

---

## 12. 调用示例

### 12.1 cURL

```bash
# 1. 发现设备
curl -X POST http://localhost:8080/api/v1/devices/discover

# 2. 添加设备
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"name":"HP 750","ip_address":"192.168.3.11","protocol":"escl"}'

# 3. 创建扫描任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "device_id":"DEVICE_ID_HERE",
    "settings":{"resolution":300,"color_mode":"Color","format":"JPEG"}
  }'

# 4. 查询进度
curl http://localhost:8080/api/v1/tasks/TASK_ID_HERE/progress

# 5. 下载文件
curl -o page.jpg http://localhost:8080/api/v1/files/FILE_ID_HERE/download
```

### 12.2 Python

```python
import requests

BASE_URL = "http://localhost:8080/api/v1"

# 创建扫描任务
response = requests.post(f"{BASE_URL}/tasks", json={
    "device_id": "DEVICE_ID_HERE",
    "settings": {
        "resolution": 300,
        "color_mode": "Color",
        "format": "JPEG",
        "input_source": "Feeder"
    }
})
task = response.json()["data"]
task_id = task["id"]

# 轮询进度
import time
while True:
    progress = requests.get(f"{BASE_URL}/tasks/{task_id}/progress").json()["data"]
    print(f"进度: {progress['progress']}%")
    if progress["status"] in ("completed", "failed"):
        break
    time.sleep(2)

# 下载文件
files = requests.get(f"{BASE_URL}/files", params={"task_id": task_id}).json()["data"]["list"]
for f in files:
    content = requests.get(f"{BASE_URL}/files/{f['id']}/download").content
    with open(f["filename"], "wb") as fp:
        fp.write(content)
```

### 12.3 JavaScript / Node.js

```javascript
const BASE_URL = 'http://localhost:8080/api/v1'

// 创建任务
const res = await fetch(`${BASE_URL}/tasks`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    device_id: 'DEVICE_ID_HERE',
    settings: { resolution: 300, color_mode: 'Color', format: 'JPEG' }
  })
})
const task = (await res.json()).data

// 轮询进度
const poll = async () => {
  const r = await fetch(`${BASE_URL}/tasks/${task.id}/progress`)
  const progress = (await r.json()).data
  console.log(`进度: ${progress.progress}%`)
  if (progress.status === 'completed' || progress.status === 'failed') return
  setTimeout(poll, 2000)
}
poll()
```

### 12.4 Go

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

func main() {
    body, _ := json.Marshal(map[string]interface{}{
        "device_id": "DEVICE_ID_HERE",
        "settings": map[string]interface{}{
            "resolution": 300, "color_mode": "Color", "format": "JPEG",
        },
    })
    resp, _ := http.Post(
        "http://localhost:8080/api/v1/tasks",
        "application/json", bytes.NewBuffer(body),
    )
    defer resp.Body.Close()

    var result struct {
        Code int `json:"code"`
        Data struct{ ID string `json:"id"` } `json:"data"`
    }
    json.NewDecoder(resp.Body).Decode(&result)
    fmt.Println("任务ID:", result.Data.ID)
}
```

---

## 13. 平板扫描仪（HP 4530等）适配说明

适用于无自动输稿器（ADF）的平板扫描设备。

### 13.1 设备识别

通过 `GET /devices/{id}` 响应的 `capabilities` 字段中的 `supports_adf` 判断：

```json
{
  "data": {
    "capabilities": "{\"supports_adf\": false, \"make_and_model\": \"HP DeskJet 4530\"}"
  }
}
```

设备列表中 `扫描方式` 列显示 "仅平板" 供识别。

### 13.2 扫描行为差异

| 场景 | ADF设备（如HP 750） | 纯平板设备（如HP 4530） |
|------|-------------------|----------------------|
| 输入源 | `Feeder`（多页） | `Platen`（单页，强制） |
| 纸张检测 | ADF -> 自动触发 | 无检测，手动放置后开始 |
| 最大页数 | 多页（35-100页） | 单页（1页） |
| 创任务设置 | `settings.input_source` 可选 | 自动强制为 Platen |

### 13.3 前端交互差异

- 扫描对话框不显示"输入源"选项
- 扫描按钮文案调整为"开始平板扫描"
- 创建任务提示放置纸张后点击开始

### 13.4 API兼容性

平板设备使用完全相同的API端点，`POST /tasks` 时的请求体无需特殊处理，但 `settings.input_source` 会被后端自动覆盖为 `"Platen"`，后端忽略该字段。

---

## 附录：接口速查表

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/devices` | 设备列表 |
| POST | `/devices` | 添加设备 |
| POST | `/devices/discover` | 发现设备 |
| GET | `/devices/{id}` | 设备详情 |
| PUT | `/devices/{id}` | 更新设备 |
| DELETE | `/devices/{id}` | 删除设备 |
| GET | `/devices/{id}/status` | 设备状态 |
| POST | `/devices/{id}/connect` | 连接设备 |
| POST | `/devices/{id}/disconnect` | 断开设备 |
| GET | `/tasks` | 任务列表 |
| POST | `/tasks` | 创建任务 |
| GET | `/tasks/{id}` | 任务详情 |
| DELETE | `/tasks/{id}` | 取消任务 |
| GET | `/tasks/{id}/progress` | 任务进度 |
| GET | `/files` | 文件列表 |
| GET | `/files/{id}` | 文件详情 |
| GET | `/files/{id}/download` | 下载文件 |
| DELETE | `/files/{id}` | 删除文件 |
| POST | `/files/batch/download` | 批量下载 |
| POST | `/files/batch/delete` | 批量删除 |
| GET | `/system/status` | 系统状态 |
| GET | `/system/logs` | 审计日志 |
| GET | `/system/config` | 系统配置 |
| PUT | `/system/config` | 更新配置 |
| WS | `/ws` | 实时通知 |

---

## 变更记录

| 版本 | 日期 | 变更 |
|------|------|------|
| v1.0.0 | 2026-06-11 | 初始版本，定义全部核心接口 |

---

**文档维护说明**：本接口文档与代码同步演进。新增或修改接口时，须同步更新本文档及 `docs/openapi.yaml`。
