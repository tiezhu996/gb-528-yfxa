package model

import "time"

type RiggingDevice struct {
	ID           uint    `gorm:"primaryKey"`
	DeviceCode   string  `gorm:"size:48;uniqueIndex;not null"`
	Name         string  `gorm:"size:120;not null"`
	DeviceType   string  `gorm:"size:32;not null"`
	MaxLoadKG    float64 `gorm:"not null;check:chk_rigging_device_load,max_load_kg > 0"`
	MaxSpeedMS   float64 `gorm:"not null;check:chk_rigging_device_speed,max_speed_ms > 0"`
	TravelMinM   float64 `gorm:"not null"`
	TravelMaxM   float64 `gorm:"not null"`
	SafetyZone   string  `gorm:"size:64;not null;index"`
	DeviceStatus string  `gorm:"size:24;not null;check:chk_rigging_device_status,device_status IN ('available','inspection_hold','retired')"`
	Version      uint    `gorm:"not null;default:1"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (RiggingDevice) TableName() string { return "rigging_devices" }
