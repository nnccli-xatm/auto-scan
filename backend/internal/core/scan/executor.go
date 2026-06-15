// 扫描执行器
// 负责任务的实际扫描执行，包括设备通信、图像下载、格式转换

package scan

import (
	"auto-scan/internal/core/device"
	"auto-scan/internal/data/models"
	"auto-scan/pkg/logger"
	"auto-scan/pkg/utils"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Executor 扫描执行器
type Executor struct {
	logger *logger.Logger
}

// NewExecutor 创建执行器
func NewExecutor(log *logger.Logger) *Executor {
	return &Executor{logger: log}
}

// ExecuteTask 执行任务扫描
func (e *Executor) ExecuteTask(ctx context.Context, task *models.ScanTask, dev *models.Device, storagePath string, progressCallback func(progress int)) (*models.ScanResult, error) {
	// 创建设备客户端
	client := device.NewESCLClient(dev.IPAddress, 0)

	// 等待ADF就绪
	if err := e.waitForADF(ctx, client); err != nil {
		return nil, utils.WrapError(utils.ErrCodeDeviceError, err, "ADF not ready")
	}

	// 解析扫描设置
	settings := models.DefaultScanSettings
	if task.Settings != "" {
		// 解析JSON设置
		// TODO: 实现JSON解析
	}

	// 创建eSCL设置
	esclSettings := device.ScanSettings{
		Version:       "2.63",
		Intent:        "Document",
		InputSource:   settings.InputSource,
		ColorMode:     settings.ColorMode,
		XResolution:   settings.Resolution,
		YResolution:   settings.Resolution,
		DocumentFormat: "image/jpeg",
	}

	// 创建扫描任务
	jobURI, err := client.CreateScanJob(ctx, esclSettings)
	if err != nil {
		return nil, utils.WrapError(utils.ErrCodeTaskCreateFailed, err, "failed to create scan job")
	}

	e.logger.Info("Created scan job: %s", jobURI)

	// 下载扫描结果
	files := []string{}
	pageCount := 0
	var totalFileSize int64 = 0

	for pageNum := 1; pageNum <= 100; pageNum++ { // 最多100页
		// 检查上下文取消
		select {
		case <-ctx.Done():
			client.DeleteJob(ctx, jobURI)
			return nil, utils.NewError(utils.ErrCodeTaskCancelled, "task cancelled")
		default:
		}

		// 尝试获取下一页
		reader, err := client.GetNextDocument(ctx, jobURI)
		if err != nil {
			// 没有更多页面
			break
		}

		// 保存文件
		filename := fmt.Sprintf("page_%03d.jpg", pageNum)
		filepath := filepath.Join(storagePath, filename)

		size, err := e.saveFile(reader, filepath)
		if err != nil {
			return nil, utils.WrapError(utils.ErrCodeFileUploadFailed, err, "failed to save file")
		}

		files = append(files, filename)
		pageCount++
		totalFileSize += size

		// 更新进度
		progress := (pageCount * 100) / (pageCount + 1) // 估算进度
		if progressCallback != nil {
			progressCallback(progress)
		}

		e.logger.Info("Downloaded page %d: %s (%d bytes)", pageNum, filename, size)
	}

	// 清理Job
	client.DeleteJob(ctx, jobURI)

	if pageCount == 0 {
		return nil, utils.NewError(utils.ErrCodeTaskFailed, "no pages scanned")
	}

	return &models.ScanResult{
		Files:       files,
		StoragePath: storagePath,
		TotalPages:  pageCount,
		FileSize:    totalFileSize,
	}, nil
}

// waitForADF 等待ADF就绪
func (e *Executor) waitForADF(ctx context.Context, client *device.ESCLClient) error {
	// 最多等待30秒
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ADF")
		default:
		}

		loaded, err := client.CheckADFLoaded(ctx)
		if err != nil {
			return err
		}

		if loaded {
			return nil
		}

		time.Sleep(1 * time.Second)
	}
}

// saveFile 保存文件
func (e *Executor) saveFile(reader io.ReadCloser, filepath string) (int64, error) {
	defer reader.Close()

	// 确保目录存在
	dir := filepath[:len(filepath)-len(filepath)]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// 写入内容
	size, err := io.Copy(file, reader)
	if err != nil {
		return 0, err
	}

	return size, nil
}