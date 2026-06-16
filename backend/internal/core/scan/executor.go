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
// 支持ADF输稿器和平板扫描两种输入源
// - ADF设备（如HP 750）：检测到纸张后自动连续扫描多页
// - 纯平板设备（如HP 4530）：单页扫描
func (e *Executor) ExecuteTask(ctx context.Context, task *models.ScanTask, dev *models.Device, storagePath string, progressCallback func(progress int)) (*models.ScanResult, error) {
	// 创建设备客户端
	client := device.NewESCLClient(dev.IPAddress, 0)

	// 获取设备能力，判断是否支持ADF
	supportsADF, err := client.SupportsADFQuery(ctx)
	if err != nil {
		return nil, utils.WrapError(utils.ErrCodeDeviceError, err, "failed to query device capabilities")
	}

	// 解析扫描设置
	settings := models.DefaultScanSettings
	if task.Settings != "" {
		// TODO: 实现JSON解析
	}

	// 强制修正输入源：
	// - 纯平板设备 → InputSource固定为Platen
	// - 带ADF设备 → 使用任务请求中指定的输入源（默认Feeder）
	inputSource := settings.InputSource
	if !supportsADF {
		inputSource = "Platen" // HP 4530等平板设备强制用Platen
	}

	// 对于平板设备，不等待ADF，等待用户放好纸张后执行单页扫描
	if supportsADF && (inputSource == "Feeder") {
		if err := e.waitForADF(ctx, client); err != nil {
			return nil, utils.WrapError(utils.ErrCodeDeviceError, err, "ADF not ready")
		}
	} else if inputSource == "Platen" {
		e.logger.Info("Platen scan mode（无ADF依赖，直接创建扫描任务）")
	}

	// 创建eSCL设置
	esclSettings := device.ScanSettings{
		Version:       "2.63",
		Intent:        "Document",
		InputSource:   inputSource, // 已根据设备类型修正
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

	maxPages := 1
	if inputSource == "Feeder" {
		maxPages = 100 // ADF连续扫描最多100页
	}

	for pageNum := 1; pageNum <= maxPages; pageNum++ {
		// 检查上下文取消
		select {
		case <-ctx.Done():
			client.DeleteJob(ctx, jobURI)
			return nil, utils.NewError(utils.ErrCodeTaskCancelled, "task cancelled")
		default:
		}

		if pageNum > 1 {
			// ADF连续模式：短暂等待下一张纸送入
			time.Sleep(1 * time.Second)
		}

		// 尝试获取下一页
		reader, err := client.GetNextDocument(ctx, jobURI)
		if err != nil {
			if pageNum == 1 {
				return nil, utils.WrapError(utils.ErrCodeFileDownloadFailed, err, "failed to get scan result")
			}
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
		progress := (pageCount * 100) / (pageCount + 1)
		if progressCallback != nil {
			progressCallback(progress)
		}

		e.logger.Info("Downloaded page %d: %s (%d bytes)", pageNum, filename, size)

		// 平板模式只扫描一页
		if inputSource == "Platen" {
			break
		}
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