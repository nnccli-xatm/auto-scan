package routes

import (
	"auto-scan/internal/api/handlers"
	"auto-scan/internal/core/scan"
	"auto-scan/internal/data/repository"
	"auto-scan/internal/service/deviceservice"
	"auto-scan/internal/service/fileservice"
	"auto-scan/internal/service/systemservice"
	"auto-scan/internal/service/taskservice"
	"auto-scan/pkg/config"
	"auto-scan/pkg/logger"
	"context"
	"database/sql"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func SetupRouter(db *sql.DB, log *logger.Logger, cfg *config.Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.Use(requestLogger(log))

	// 初始化数据库层
	repoDB := &repository.DB{DB: db}
	deviceRepo := repository.NewDeviceRepository(repoDB)
	taskRepo := repository.NewTaskRepository(repoDB)
	fileRepo := repository.NewFileRepository(repoDB)

	// 初始化Service层
	deviceService, _ := deviceservice.NewDeviceService(deviceRepo, log)
	scanExecutor := scan.NewExecutor(log)
	taskService := taskservice.NewTaskServiceWithExecutor(taskRepo, deviceRepo, scanExecutor, fileRepo, log)

	// 启动任务调度器（后台执行扫描）
	taskService.StartScheduler(context.Background())

	cfgManager, _ := config.NewManager("config.yaml")
	fileService := fileservice.NewFileService(fileRepo)

	systemService := systemservice.NewSystemService(cfgManager, deviceRepo, taskRepo, fileRepo, log)

	// 初始化Handler
	h := handlers.NewHandler(deviceService, taskService, systemService, fileService)

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 设备管理
		devices := v1.Group("/devices")
		{
			devices.GET("", h.ListDevices)
			devices.POST("", h.CreateDevice)
			devices.POST("/discover", h.DiscoverDevices)
			devices.GET("/:id", h.GetDevice)
			devices.PUT("/:id", h.UpdateDevice)
			devices.DELETE("/:id", h.DeleteDevice)
			devices.GET("/:id/status", h.GetDeviceStatus)
		}

		// 任务管理
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", h.ListTasks)
			tasks.POST("", h.CreateTask)
			tasks.GET("/:id", h.GetTask)
			tasks.DELETE("/:id", h.CancelTask)
			tasks.GET("/:id/progress", h.GetTaskProgress)
		}

		// 文件管理
		files := v1.Group("/files")
		{
			files.GET("", h.ListFiles)
			files.GET("/:id", h.GetFile)
			files.DELETE("/:id", h.DeleteFile)
			files.GET("/:id/download", h.DownloadFile)
			files.POST("/batch-download", h.BatchDownloadFiles)
			files.POST("/batch-delete", h.BatchDeleteFiles)
		}

		// 系统管理
		system := v1.Group("/system")
		{
			system.GET("/status", h.GetSystemStatus)
			system.GET("/logs", h.GetSystemLogs)
			system.GET("/config", h.GetSystemConfig)
			system.PUT("/config", h.UpdateSystemConfig)
		}
	}

	return router
}

func requestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		c.Next()
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		if raw != "" {
			path = path + "?" + raw
		}
		log.Info("[%s] %s %s %d %v", clientIP, method, path, statusCode, latency)
	}
}
