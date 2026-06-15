# Auto Scan Frontend

扫描设备自动任务系统前端 - Vue3 + TypeScript

## 技术栈

- **框架**: Vue 3 (Composition API)
- **语言**: TypeScript 5.0+
- **UI库**: Element Plus
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **HTTP**: Axios
- **构建工具**: Vite 5

## 项目结构

```
frontend/
├── src/
│   ├── api/              # API接口层
│   │   ├── client.ts     # Axios实例
│   │   ├── device.ts     # 设备API
│   │   ├── task.ts       # 任务API
│   │   ├── file.ts       # 文件API
│   │   ├── system.ts     # 系统API
│   │   └── types.ts      # 类型定义
│   ├── components/       # 组件
│   │   └── Layout.vue    # 主布局
│   ├── stores/           # 状态管理
│   │   ├── device.ts     # 设备状态
│   │   ├── task.ts       # 任务状态
│   │   └── system.ts     # 系统状态
│   ├── views/            # 页面
│   │   ├── Dashboard.vue     # 仪表板
│   │   ├── DeviceList.vue    # 设备列表
│   │   ├── DeviceDetail.vue  # 设备详情
│   │   ├── TaskList.vue      # 任务管理
│   │   ├── FileBrowser.vue   # 文件浏览
│   │   ├── Logs.vue          # 系统日志
│   │   └── Settings.vue      # 系统设置
│   ├── router/           # 路由配置
│   ├── style.css         # 全局样式
│   ├── App.vue           # 根组件
│   └── main.ts           # 入口
├── index.html
├── package.json
├── vite.config.ts
└── tsconfig.json
```

## 快速开始

### 安装依赖

```bash
npm install
```

### 开发模式

```bash
npm run dev
```

开发服务器运行在 `http://localhost:3000`，自动代理API请求到后端 `http://localhost:8080`。

### 构建生产版本

```bash
npm run build
```

### 预览构建结果

```bash
npm run preview
```

## 功能模块

### 仪表板 (Dashboard)
- 设备总数、在线设备统计
- 进行中任务、文件数量统计
- 设备概览列表
- 存储使用情况
- 最近任务列表

### 设备管理 (Devices)
- 设备列表（搜索、过滤）
- 自动发现设备（mDNS）
- 手动添加设备
- 设备连接/断开
- 设备详情查看
- 启动扫描任务

### 任务管理 (Tasks)
- 任务列表（状态过滤）
- 任务进度实时显示
- 任务取消
- 任务状态跟踪

### 文件管理 (Files)
- 扫描文件浏览
- 图片预览
- 单文件下载
- 批量下载（ZIP）
- 单文件/批量删除

### 系统日志 (Logs)
- 日志查询（级别、时间范围）
- 日志详情查看
- 设备/任务关联

### 系统设置 (Settings)
- 服务器配置
- 扫描参数配置
- 设备监控设置
- 日志级别配置
- 系统信息查看
- 存储信息查看

## API对接

前端通过Axios与后端RESTful API通信，所有接口定义遵循OpenAPI规范（`../docs/openapi.yaml`）。

### 请求拦截器
- 自动添加JWT Token
- 统一请求头

### 响应拦截器
- 统一错误处理
- Token过期处理
- 友好的错误提示

## 开发说明

- 所有页面使用Vue 3 Composition API (`<script setup>`)
- 状态管理使用Pinia（Composition风格）
- 类型安全：所有API响应都有TypeScript类型定义
- 响应式设计：支持不同屏幕尺寸

## 环境变量

- `VITE_API_BASE_URL`: API基础URL
  - 开发环境: `http://localhost:8080/api/v1`
  - 生产环境: `/api/v1`（相对路径，通过Nginx代理）
