# Auto Scan - HP Smart Tank 750 ADF 自动扫描程序

自动检测HP Smart Tank 750扫描仪ADF（自动输稿器）中的纸张，并自动扫描保存为JPG图片。

## 功能特点

- ✅ **自动检测**：实时监测ADF纸张状态
- ✅ **连续扫描**：支持多张纸连续自动扫描（最多35张）
- ✅ **单页输出**：每页生成单独的JPG文件
- ✅ **高质量**：300dpi彩色扫描
- ✅ **智能命名**：按时间创建文件夹，文件名包含页码

## 文件说明

- `hp_scanner_daemon.py` - 核心守护程序（Python）
- `scanner_control.sh` - 控制脚本（启动/停止/状态查看）
- `README.md` - 本说明文件

## 安装依赖

```bash
# macOS
brew install poppler  # PDF处理依赖
pip3 install pdf2image requests pillow
```

## 使用方法

### 1. 启动守护程序

```bash
./scanner_control.sh start
```

或直接运行：
```bash
python3 hp_scanner_daemon.py
```

### 2. 放入纸张

- 打开扫描仪顶部ADF盖子
- 纸张面朝上，顶部朝里放入
- 最多可放35张

### 3. 自动扫描

程序会自动检测纸张并开始扫描，每页保存为单独的JPG文件。

### 4. 查看结果

扫描文件保存在 `~/测试图片/<时间>/` 目录下，格式为：
- `page_001.jpg` - 单页文件
- `page_001_01.jpg, page_001_02.jpg...` - 多页拆分

### 5. 控制命令

```bash
./scanner_control.sh status   # 查看状态
./scanner_control.sh stop     # 停止守护程序
./scanner_control.sh logs     # 查看实时日志
./scanner_control.sh restart  # 重启
```

## 配置

编辑 `hp_scanner_daemon.py` 修改以下配置：

```python
SCANNER_IP = "192.168.3.11"           # 扫描仪IP地址
OUTPUT_BASE = Path("~/测试图片")       # 输出目录
POLL_INTERVAL = 2                      # 检测间隔（秒）
RESOLUTION = 300                       # 扫描分辨率（dpi）
```

## 日志文件

日志保存在：`/tmp/hp_scanner_daemon.log`

查看日志：
```bash
tail -f /tmp/hp_scanner_daemon.log
```

## 技术细节

- **协议**：eSCL (AirPrint Scan)
- **接口**：HTTP API (192.168.3.11)
- **格式**：JPEG (quality=95)
- **色彩**：RGB24 真彩色

## 故障排除

### 扫描仪未响应
- 确认扫描仪IP地址正确
- 检查网络连接
- 查看日志：`./scanner_control.sh logs`

### 503错误
- 扫描仪正在处理其他任务，等待10-30秒
- 重启扫描仪电源

### 无纸张检测
- 确认纸张面朝上放置
- 检查ADF盖子是否完全闭合
- 纸张不要太厚或太薄

## 许可证

MIT License
