package models

import "time"

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

// Vendor 厂商常量
const (
	VendorHP       = "HP"
	VendorCanon    = "Canon"
	VendorRicoh    = "Ricoh"
	VendorFujitsu  = "Fujitsu"
	VendorBrother  = "Brother"
	VendorEpson    = "Epson"
	VendorOther    = "Other"
)