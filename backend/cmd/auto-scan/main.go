// Auto Scan - 扫描设备自动任务系统
// 后端服务主入口

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"auto-scan/internal/api/routes"
	"auto-scan/internal/data/repository"
	"auto-scan/pkg/config"
	"auto-scan/pkg/logger"
)

func main() {
	// 命令行参数
	var (
		configPath = flag.String("config", "config.yaml", "配置文件路径")
		port       = flag.String("port", "8080", "服务端口")
	)
	flag.Parse()

	// 初始化日志
	log := logger.NewLogger()
	log.Info("Auto Scan starting...")

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("Failed to load config:", err)
		os.Exit(1)
	}

	// 初始化数据库
	db, err := repository.Initialize(cfg.Database.Path)
	if err != nil {
		log.Error("Failed to initialize database:", err)
		os.Exit(1)
	}
	defer db.Close()

	// 初始化路由
	router := routes.SetupRouter(db, log)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: router,
	}

	// 优雅关闭
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("Server forced to shutdown:", err)
		}

		log.Info("Server exited")
	}()

	// 启动服务器
	log.Info("Server starting on port " + *port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("Failed to start server:", err)
		os.Exit(1)
	}
}
