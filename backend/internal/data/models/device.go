package models

import (
	"strings"
	"time"
)

// Device 设备模型
type Device struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	Protocol     string    `json:"protocol" db:"protocol"` // escl, wsd
	Model        string    `json:"model" db:"model"`
	Vendor       string    `json:"vendor" db:"vendor"`
	Status       string    `json:"status" db:"status"` // online, offline, busy, error
	Capabilities string    `json:"capabilities" db:"capabilities"` // JSON
	Config       string    `json:"config" db:"config"`             // JSON
	LastSeen     time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// DeviceStatus 设备状态常量
const (
	DeviceStatusOnline  = "online"
	DeviceStatusOffline = "offline"
	DeviceStatusBusy    = "busy"
	DeviceStatusError   = "error"
)

// Protocol 协议类型常量
const (
	ProtocolESCL = "escl"
	ProtocolWSD  = "wsd"
)

// DeviceCapabilities 设备能力（JSON序列化/反序列化的结构）
type DeviceCapabilities struct {
	SupportsADF  bool     `json:"supports_adf"`
	FeederCapacity int    `json:"feeder_capacity,omitempty"`
	Resolutions  []int    `json:"resolutions,omitempty"`
	ColorModes   []string `json:"color_modes,omitempty"`
}

const (
	VendorHP       = "HP"
	VendorCanon    = "Canon"
	VendorRicoh    = "Ricoh"
	VendorFujitsu  = "Fujitsu"
	VendorBrother  = "Brother"
	VendorEpson    = "Epson"
	VendorOther    = "Other"
)

// NormalizeVendor 规范化厂商字段，不匹配时返回Other
func NormalizeVendor(v string) string {
	valid := map[string]bool{
		"HP": true, "Canon": true, "Ricoh": true,
		"Fujitsu": true, "Brother": true, "Epson": true,
	}
	if valid[v] {
		return v
	}
	// 部分匹配
	upper := strings.ToUpper(v)
	if strings.Contains(upper, "HP") {
		return "HP"
	}
	if strings.Contains(upper, "CANON") {
		return "Canon"
	}
	if strings.Contains(upper, "RICOH") {
		return "Ricoh"
	}
	if strings.Contains(upper, "FUJITSU") {
		return "Fujitsu"
	}
	if strings.Contains(upper, "BROTHER") {
		return "Brother"
	}
	if strings.Contains(upper, "EPSON") {
		return "Epson"
	}
	return "Other"
}