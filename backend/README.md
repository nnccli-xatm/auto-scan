# Auto Scan Backend

扫描设备自动任务系统后端服务

## 技术栈

- **语言**: Go 1.22+
- **Web框架**: Gin
- **数据库**: SQLite3 (WAL模式)
- **服务发现**: Zeroconf (mDNS)
- **日志**: Logrus

## 项目结构

```
backend/
├── cmd/auto-scan/          # 主入口
├── internal/
│   ├── api/                # API层
│   │   ├── handlers/       # HTTP处理器
│   │   ├── middleware/     # 中间件
│   │   └── routes/         # 路由配置
│   ├── core/               # 核心业务逻辑
│   │   ├── device/         # 设备管理 (eSCL协议)
│   │   ├── scan/           # 扫描任务
│   │   └── storage/        # 存储管理
│   ├── data/               # 数据访问层
│   │   ├── models/         # 数据模型
│   │   └── repository/     # 数据仓库
│   └── service/            # 业务服务层
├── pkg/                    # 公共包
│   ├── config/             # 配置管理
│   ├── logger/             # 日志工具
│   └── utils/              # 工具函数
└── test/                   # 测试
```

## 快速开始

### 安装依赖

```bash
go mod download
```

### 运行服务

```bash
go run cmd/auto-scan/main.go
```

### 构建

```bash
# Linux
go build -o auto-scan cmd/auto-scan/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o auto-scan.exe cmd/auto-scan/main.go
```

## API文档

详见 `../docs/openapi.yaml`

## 数据库

数据库Schema: `../docs/schema.sql`

```bash
# 创建数据库
sqlite3 auto-scan.db < ../docs/schema.sql

# 启用WAL模式
sqlite3 auto-scan.db "PRAGMA journal_mode=WAL;"
```

## Week 1 开发计划

- [x] 项目脚手架搭建
- [x] eSCL协议客户端
- [x] mDNS设备发现
- [ ] API路由实现
- [ ] 数据库集成
- [ ] 设备管理CRUD
- [ ] 扫描任务基础流程
