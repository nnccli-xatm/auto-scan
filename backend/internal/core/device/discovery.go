// 设备发现服务
// 支持mDNS自动发现和手动添加

package device

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
)

// DiscoveryService 设备发现服务
type DiscoveryService struct {
	resolver  *zeroconf.Resolver
	devices   chan *DiscoveredDevice
	stopChan  chan struct{}
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.Mutex
}

// DiscoveredDevice 发现的设备信息
type DiscoveredDevice struct {
	Name     string
	Host     string
	IP       string
	Port     int
	Protocol string // escl, wsd
	Model    string
	Vendor   string
}

// NewDiscoveryService 创建设备发现服务
func NewDiscoveryService() (*DiscoveryService, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create zeroconf resolver: %w", err)
	}

	return &DiscoveryService{
		resolver: resolver,
		devices:  make(chan *DiscoveredDevice, 100),
		stopChan: make(chan struct{}),
	}, nil
}

// Start 开始发现服务
func (d *DiscoveryService) Start() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.isRunning {
		return fmt.Errorf("discovery service already running")
	}

	d.isRunning = true
	d.wg.Add(3)
	go d.discoverService("_uscan._tcp", "escl")
	go d.discoverService("_scanner._tcp", "escl")
	go d.discoverService("_ipp._tcp", "escl")

	return nil
}

// Stop 停止发现服务
func (d *DiscoveryService) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.isRunning {
		return
	}

	close(d.stopChan)
	d.wg.Wait()
	d.isRunning = false
}

// DiscoverOnce 执行一次发现
func (d *DiscoveryService) DiscoverOnce(ctx context.Context, timeout time.Duration) ([]*DiscoveredDevice, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	devices := []*DiscoveredDevice{}
	deviceChan := make(chan *DiscoveredDevice, 100)

	go d.discoverWithContext(ctx, deviceChan)

	for {
		select {
		case <-ctx.Done():
			return devices, nil
		case dev := <-deviceChan:
			if dev != nil {
				devices = append(devices, dev)
			}
		}
	}
}

func (d *DiscoveryService) discoverService(service, protocol string) {
	defer d.wg.Done()
	entries := make(chan *zeroconf.ServiceEntry, 100)

	go func() {
		for {
			select {
			case <-d.stopChan:
				return
			case entry := <-entries:
				if entry == nil {
					continue
				}
				ip := d.resolveIP(entry)
				if ip == "" {
					continue
				}
				vendor, model := d.parseDeviceInfo(entry)
				device := &DiscoveredDevice{
					Name: entry.Instance, Host: entry.HostName,
					IP: ip, Port: entry.Port, Protocol: protocol,
					Model: model, Vendor: vendor,
				}
				select {
				case d.devices <- device:
				case <-d.stopChan:
					return
				}
			}
		}
	}()

	d.resolver.Browse(context.Background(), service, "local.", entries)
	<-d.stopChan
}

func (d *DiscoveryService) discoverWithContext(ctx context.Context, outChan chan<- *DiscoveredDevice) {
	services := []struct{ service, protocol string }{
		{"_uscan._tcp", "escl"}, {"_scanner._tcp", "escl"}, {"_ipp._tcp", "escl"},
	}

	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(s, p string) {
			defer wg.Done()
			entries := make(chan *zeroconf.ServiceEntry, 10)
			go func() {
				for entry := range entries {
					if entry == nil {
						continue
					}
					ip := d.resolveIP(entry)
					if ip == "" {
						continue
					}
					vendor, model := d.parseDeviceInfo(entry)
					device := &DiscoveredDevice{
						Name: entry.Instance, Host: entry.HostName,
						IP: ip, Port: entry.Port, Protocol: p,
						Model: model, Vendor: vendor,
					}
					select {
					case outChan <- device:
					case <-ctx.Done():
						return
					}
				}
			}()
			browseCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			d.resolver.Browse(browseCtx, s, "local.", entries)
		}(svc.service, svc.protocol)
	}
	wg.Wait()
	close(outChan)
}

func (d *DiscoveryService) resolveIP(entry *zeroconf.ServiceEntry) string {
	for _, addr := range entry.AddrIPv4 {
		return addr.String()
	}
	for _, addr := range entry.AddrIPv6 {
		return addr.String()
	}
	if entry.HostName != "" {
		ips, err := net.LookupHost(entry.HostName)
		if err == nil && len(ips) > 0 {
			return ips[0]
		}
	}
	return ""
}

func (d *DiscoveryService) parseDeviceInfo(entry *zeroconf.ServiceEntry) (vendor, model string) {
	name := entry.Instance
	vendors := []string{"HP", "Canon", "Ricoh", "Fujitsu", "Brother", "Epson"}
	for _, v := range vendors {
		if strings.Contains(strings.ToUpper(name), strings.ToUpper(v)) {
			vendor = v
			break
		}
	}
	for _, txt := range entry.Text {
		if strings.HasPrefix(txt, "ty=") {
			model = strings.TrimPrefix(txt, "ty=")
		}
		if strings.HasPrefix(txt, "product=") {
			model = strings.TrimPrefix(txt, "product=")
		}
	}
	if model == "" {
		model = name
	}
	return vendor, model
}

func (d *DiscoveryService) GetDevicesChannel() <-chan *DiscoveredDevice {
	return d.devices
}

func (d *DiscoveryService) IsRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.isRunning
}

func ManualDiscovery(ip string, port int) (*DiscoveredDevice, error) {
	client := NewESCLClient(ip, port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get capabilities: %w", err)
	}

	vendor := "Unknown"
	vendors := []string{"HP", "Canon", "Ricoh", "Fujitsu", "Brother", "Epson"}
	for _, v := range vendors {
		if strings.Contains(strings.ToUpper(caps.MakeAndModel), strings.ToUpper(v)) {
			vendor = v
			break
		}
	}

	return &DiscoveredDevice{
		Name: caps.MakeAndModel, IP: ip, Port: port,
		Protocol: "escl", Model: caps.MakeAndModel, Vendor: vendor,
	}, nil
}
