package routes

import (
	"auto-scan/internal/api/handlers"
	"auto-scan/pkg/logger"
	"database/sql"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter 配置路由
func SetupRouter(db *sql.DB, log *logger.Logger) *gin.Engine {
	// 设置Gin模式
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// 全局中间件
	router.Use(gin.Recovery())
	router.Use(cors.Default())
	router.Use(requestLogger(log))

	// API v1 路由组
	v1 := router.Group("/api/v1")
	{
		// 设备管理
		deviceHandler := handlers.NewDeviceHandler()
		devices := v1.Group("/devices")
		{
			devices.GET("", deviceHandler.ListDevices)
			devices.POST("", deviceHandler.CreateDevice)
			devices.POST("/discover", deviceHandler.DiscoverDevices)
			devices.GET("/:id", deviceHandler.GetDevice)
			devices.PUT("/:id", deviceHandler.UpdateDevice)
			devices.DELETE("/:id", deviceHandler.DeleteDevice)
			devices.GET("/:id/status", deviceHandler.GetDeviceStatus)
		}

		// TODO: 任务管理
		tasks := v1.Group("/tasks")
		{
			tasks.GET("")
			tasks.POST("")
			tasks.GET("/:id")
			tasks.DELETE("/:id")
			tasks.GET("/:id/progress")
		}

		// TODO: 文件管理
		files := v1.Group("/files")
		{
			files.GET("")
			files.GET("/:id")
			files.DELETE("/:id")
			files.GET("/:id/download")
			files.POST("/batch-download")
			files.POST("/batch-delete")
		}

		// TODO: 系统管理
		system := v1.Group("/system")
		{
			system.GET("/status")
			system.GET("/logs")
			system.GET("/config")
			system.PUT("/config")
		}
	}

	return router
}

// requestLogger 请求日志中间件
func requestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info("[%s] %s %s %d %v",
			clientIP,
			method,
			path,
			statusCode,
			latency,
		)
	}
}
