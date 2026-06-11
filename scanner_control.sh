#!/bin/bash
# HP Smart Tank 750 ADF 扫描仪控制脚本

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DAEMON_SCRIPT="$SCRIPT_DIR/hp_scanner_daemon.py"
PID_FILE="/tmp/hp_scanner_daemon.pid"
LOG_FILE="/tmp/hp_scanner_daemon.log"

check_python() {
    if ! command -v python3 &> /dev/null; then
        echo "错误: 未找到 Python3"
        exit 1
    fi
}

start() {
    check_python

    if [ -f "$PID_FILE" ] && kill -0 $(cat "$PID_FILE") 2>/dev/null; then
        echo "守护程序已经在运行 (PID: $(cat "$PID_FILE"))"
        exit 1
    fi

    echo "正在启动 HP 扫描仪守护程序..."
    nohup python3 "$DAEMON_SCRIPT" > "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"

    sleep 1
    if kill -0 $(cat "$PID_FILE") 2>/dev/null; then
        echo "✅ 守护程序已启动 (PID: $(cat "$PID_FILE"))"
        echo "日志文件: $LOG_FILE"
        echo "输出目录: ~/测试图片/"
        echo ""
        echo "使用方法:"
        echo "  1. 将纸张放入扫描仪 ADF（自动输稿器）"
        echo "  2. 程序会自动检测并扫描所有页面"
        echo "  3. 扫描的文件会保存在 ~/测试图片/<时间>/ 目录下"
    else
        echo "❌ 启动失败，查看日志: $LOG_FILE"
        exit 1
    fi
}

stop() {
    if [ ! -f "$PID_FILE" ]; then
        echo "守护程序未运行"
        exit 1
    fi

    PID=$(cat "$PID_FILE")
    echo "正在停止守护程序 (PID: $PID)..."

    if kill "$PID" 2>/dev/null; then
        rm -f "$PID_FILE"
        echo "✅ 守护程序已停止"
    else
        echo "❌ 停止失败，进程可能已经结束"
        rm -f "$PID_FILE"
    fi
}

status() {
    if [ -f "$PID_FILE" ] && kill -0 $(cat "$PID_FILE") 2>/dev/null; then
        echo "✅ 守护程序正在运行 (PID: $(cat "$PID_FILE"))"
        echo "日志文件: $LOG_FILE"
        echo "最近日志:"
        tail -n 5 "$LOG_FILE" 2>/dev/null || echo "暂无日志"
    else
        echo "❌ 守护程序未运行"
        if [ -f "$LOG_FILE" ]; then
            echo "上次运行日志:"
            tail -n 10 "$LOG_FILE" 2>/dev/null
        fi
    fi
}

logs() {
    if [ -f "$LOG_FILE" ]; then
        echo "查看日志 (按 Ctrl+C 退出)..."
        tail -f "$LOG_FILE"
    else
        echo "日志文件不存在"
    fi
}

install() {
    echo "正在安装依赖..."
    pip3 install requests pillow --user
    chmod +x "$DAEMON_SCRIPT"
    chmod +x "$SCRIPT_DIR/scanner_control.sh"
    echo "✅ 安装完成"
}

case "${1:-}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        stop 2>/dev/null || true
        sleep 1
        start
        ;;
    status)
        status
        ;;
    logs)
        logs
        ;;
    install)
        install
        ;;
    *)
        echo "HP Smart Tank 750 ADF 扫描仪控制脚本"
        echo ""
        echo "用法: $0 {start|stop|restart|status|logs|install}"
        echo ""
        echo "命令:"
        echo "  start    - 启动守护程序"
        echo "  stop     - 停止守护程序"
        echo "  restart  - 重启守护程序"
        echo "  status   - 查看运行状态"
        echo "  logs     - 实时查看日志"
        echo "  install  - 安装依赖"
        echo ""
        echo "首次使用请先运行: $0 install"
        exit 1
        ;;
esac
