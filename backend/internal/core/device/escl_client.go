// eSCL协议客户端实现
// 参考: AirPrint Scan (eSCL)协议规范
// 支持设备发现、状态查询、扫描任务管理

package device

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ESCLClient eSCL协议客户端
type ESCLClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewESCLClient 创建eSCL客户端
func NewESCLClient(ipAddress string, port int) *ESCLClient {
	if port == 0 {
		port = 80 // 默认端口
	}

	return &ESCLClient{
		baseURL: fmt.Sprintf("http://%s:%d", ipAddress, port),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ScannerCapabilities 扫描仪能力信息
type ScannerCapabilities struct {
	XMLName     xml.Name           `xml:"ScannerCapabilities"`
	Version     string             `xml:"Version"`
	MakeAndModel string            `xml:"MakeAndModel"`
	SerialNumber string            `xml:"SerialNumber"`
	Manufacturer string            `xml:"Manufacturer"`
	Platen      PlatenCapabilities `xml:"Platen"`
	ADF         *ADFCapabilities   `xml:"Adf,omitempty"` // 指针类型，无ADF时为nil
}

// SupportsADF 检查是否支持自动输稿器
func (c *ScannerCapabilities) SupportsADF() bool {
	return c.ADF != nil
}

type PlatenCapabilities struct {
	PlatenInputCaps InputCapabilities `xml:"PlatenInputCaps"`
}

type ADFCapabilities struct {
	AdfSimplexInputCaps InputCapabilities `xml:"AdfSimplexInputCaps"`
	FeederCapacity      int               `xml:"FeederCapacity"`
}

type InputCapabilities struct {
	MinWidth      int    `xml:"MinWidth"`
	MaxWidth      int    `xml:"MaxWidth"`
	MinHeight     int    `xml:"MinHeight"`
	MaxHeight     int    `xml:"MaxHeight"`
	ColorModes    []string `xml:"SettingProfiles>SettingProfile>ColorModes>ColorMode"`
	Resolutions   []int    `xml:"SettingProfiles>SettingProfile>SupportedResolutions>DiscreteResolutions>DiscreteResolution>XResolution"`
}

// ScannerStatus 扫描仪状态
type ScannerStatus struct {
	XMLName     xml.Name `xml:"ScannerStatus"`
	Version     string   `xml:"Version"`
	State       string   `xml:"State"`        // Idle, Processing
	AdfState    string   `xml:"AdfState"`     // ScannerAdfEmpty, ScannerAdfLoaded
	Jobs        []JobInfo `xml:"Jobs>JobInfo"`
}

type JobInfo struct {
	JobURI          string `xml:"JobUri"`
	JobState        string `xml:"JobState"`        // Pending, Processing, Completed, Aborted
	ImagesCompleted int    `xml:"ImagesCompleted"`
	ImagesToTransfer int   `xml:"ImagesToTransfer"`
}

// ScanSettings 扫描设置
type ScanSettings struct {
	XMLName       xml.Name `xml:"ScanSettings"`
	Version       string   `xml:"Version"`
	Intent        string   `xml:"Intent"`        // Document, Photo
	InputSource   string   `xml:"InputSource"`   // Platen, Feeder
	ColorMode     string   `xml:"ColorMode"`     // BlackAndWhite1, Grayscale8, RGB24
	XResolution   int      `xml:"XResolution"`
	YResolution   int      `xml:"YResolution"`
	DocumentFormat string  `xml:"DocumentFormat"` // image/jpeg, application/pdf
}

// GetCapabilities 获取扫描仪能力
func (c *ESCLClient) GetCapabilities(ctx context.Context) (*ScannerCapabilities, error) {
	url := fmt.Sprintf("%s/eSCL/ScannerCapabilities", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var caps ScannerCapabilities
	if err := xml.Unmarshal(body, &caps); err != nil {
		return nil, fmt.Errorf("parse XML failed: %w", err)
	}

	return &caps, nil
}

// GetStatus 获取扫描仪状态
func (c *ESCLClient) GetStatus(ctx context.Context) (*ScannerStatus, error) {
	url := fmt.Sprintf("%s/eSCL/ScannerStatus", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var status ScannerStatus
	if err := xml.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("parse XML failed: %w", err)
	}

	return &status, nil
}

// CreateScanJob 创建扫描任务（使用原始XML字符串避免命名空间问题）
func (c *ESCLClient) CreateScanJob(ctx context.Context, settings ScanSettings) (string, error) {
	ver := settings.Version
	if ver == "" {
		ver = "2.63"
	}

	rawXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<scan:ScanSettings xmlns:scan="http://schemas.hp.com/imaging/escl/2011/05/03" xmlns:pwg="http://www.pwg.org/schemas/2010/12/sm">
  <pwg:Version>%s</pwg:Version>
  <scan:Intent>%s</scan:Intent>
  <scan:InputSource>%s</scan:InputSource>
  <scan:ColorMode>%s</scan:ColorMode>
  <scan:XResolution>%d</scan:XResolution>
  <scan:YResolution>%d</scan:YResolution>
  <pwg:DocumentFormat>%s</pwg:DocumentFormat>
</scan:ScanSettings>`,
		ver, settings.Intent, settings.InputSource, settings.ColorMode,
		settings.XResolution, settings.YResolution, settings.DocumentFormat)

	url := fmt.Sprintf("%s/eSCL/ScanJobs", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBufferString(rawXML))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 从Location头获取Job URI
	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no Location header in response")
	}

	return location, nil
}

// GetNextDocument 获取扫描的文档
func (c *ESCLClient) GetNextDocument(ctx context.Context, jobURI string) (io.ReadCloser, error) {
	if !strings.HasPrefix(jobURI, "http") {
		jobURI = c.baseURL + jobURI
	}
	url := fmt.Sprintf("%s/NextDocument", jobURI)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// DeleteJob 删除扫描任务
func (c *ESCLClient) DeleteJob(ctx context.Context, jobURI string) error {
	url := jobURI
	if !strings.HasPrefix(url, "http") {
		url = c.baseURL + url
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// CheckADFLoaded 检查ADF是否有纸（仅带ADF的设备可用）
// 无ADF设备（如HP 4530）直接返回false
func (c *ESCLClient) CheckADFLoaded(ctx context.Context) (bool, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return false, err
	}
	return status.AdfState == "ScannerAdfLoaded", nil
}

// SupportsADFQuery 通过ScannerCapabilities判定该设备是否支持ADF
func (c *ESCLClient) SupportsADFQuery(ctx context.Context) (bool, error) {
	caps, err := c.GetCapabilities(ctx)
	if err != nil {
		return false, err
	}
	return caps.SupportsADF(), nil
}

// SupportsPlaten 平板扫描是否可用（所有eSCL设备都支持平板）
func (c *ESCLClient) SupportsPlaten(ctx context.Context) (bool, error) {
	caps, err := c.GetCapabilities(ctx)
	if err != nil {
		return false, err
	}
	return caps.Platen.PlatenInputCaps.MaxWidth > 0, nil
}

// WaitForJob 等待平板扫描任务结束（非ADF等待）
func (c *ESCLClient) WaitForJob(ctx context.Context, jobURI string, timeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	downloaded := 0
	for {
		select {
		case <-ctx.Done():
			return downloaded, fmt.Errorf("timeout waiting for scan job")
		default:
		}

		// 尝试直接获取NextDocument获取图片内容
		reader, err := c.GetNextDocument(ctx, jobURI)
		if err != nil {
			// 任务还未就绪，继续等待
			time.Sleep(500 * time.Millisecond)
			continue
		}
		defer reader.Close()

		downloaded++
		// 成功后结束
		return downloaded, nil
	}
}

// WaitForADF 等待ADF加载纸张（带超时）
func (c *ESCLClient) WaitForADF(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for ADF")
		case <-ticker.C:
			loaded, err := c.CheckADFLoaded(ctx)
			if err != nil {
				return err
			}
			if loaded {
				return nil
			}
		}
	}
}
