package dto

import (
	"time"

	"stage-rigging-cue-interlock/backend/internal/model"
)

type CreateRiggingDeviceRequest struct {
	DeviceCode   string  `json:"device_code" binding:"required,min=3,max=48"`
	Name         string  `json:"name" binding:"required,min=3,max=120"`
	DeviceType   string  `json:"device_type" binding:"required,oneof=motorized_batten motorized_bridge scenic_carrier point_hoist manual_counterweight"`
	MaxLoadKG    float64 `json:"max_load_kg" binding:"required,gt=0,lte=100000"`
	MaxSpeedMS   float64 `json:"max_speed_ms" binding:"required,gt=0,lte=10"`
	TravelMinM   float64 `json:"travel_min_m" binding:"gte=-20,lte=200"`
	TravelMaxM   float64 `json:"travel_max_m" binding:"required,gte=-20,lte=200"`
	SafetyZone   string  `json:"safety_zone" binding:"required,min=2,max=64"`
	DeviceStatus string  `json:"device_status" binding:"required,oneof=available inspection_hold retired"`
}

type UpdateRiggingDeviceRequest struct {
	Name         string  `json:"name" binding:"required,min=3,max=120"`
	DeviceType   string  `json:"device_type" binding:"required,oneof=motorized_batten motorized_bridge scenic_carrier point_hoist manual_counterweight"`
	MaxLoadKG    float64 `json:"max_load_kg" binding:"required,gt=0,lte=100000"`
	MaxSpeedMS   float64 `json:"max_speed_ms" binding:"required,gt=0,lte=10"`
	TravelMinM   float64 `json:"travel_min_m" binding:"gte=-20,lte=200"`
	TravelMaxM   float64 `json:"travel_max_m" binding:"required,gte=-20,lte=200"`
	SafetyZone   string  `json:"safety_zone" binding:"required,min=2,max=64"`
	DeviceStatus string  `json:"device_status" binding:"required,oneof=available inspection_hold retired"`
	Version      uint    `json:"version" binding:"required,gte=1"`
}

type RuleReference struct {
	ID          uint   `json:"id"`
	RuleCode    string `json:"rule_code"`
	RuleType    string `json:"rule_type"`
	Severity    string `json:"severity"`
	Enabled     bool   `json:"enabled"`
	RuleVersion uint   `json:"rule_version"`
}

type RiggingDeviceResponse struct {
	ID              uint            `json:"id"`
	DeviceCode      string          `json:"device_code"`
	Name            string          `json:"name"`
	DeviceType      string          `json:"device_type"`
	MaxLoadKG       float64         `json:"max_load_kg"`
	MaxSpeedMS      float64         `json:"max_speed_ms"`
	TravelMinM      float64         `json:"travel_min_m"`
	TravelMaxM      float64         `json:"travel_max_m"`
	SafetyZone      string          `json:"safety_zone"`
	DeviceStatus    string          `json:"device_status"`
	Version         uint            `json:"version"`
	ApplicableRules []RuleReference `json:"applicable_rules"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

func RiggingDeviceFromModel(item model.RiggingDevice) RiggingDeviceResponse {
	return RiggingDeviceResponse{ID: item.ID, DeviceCode: item.DeviceCode, Name: item.Name, DeviceType: item.DeviceType, MaxLoadKG: item.MaxLoadKG, MaxSpeedMS: item.MaxSpeedMS, TravelMinM: item.TravelMinM, TravelMaxM: item.TravelMaxM, SafetyZone: item.SafetyZone, DeviceStatus: item.DeviceStatus, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, ApplicableRules: []RuleReference{}}
}
