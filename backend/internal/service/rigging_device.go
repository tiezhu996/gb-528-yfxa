package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/util"
)

type RiggingDeviceService struct {
	devices *repository.RiggingDeviceRepository
	rules   *repository.InterlockRuleRepository
}

func NewRiggingDeviceService(devices *repository.RiggingDeviceRepository, rules *repository.InterlockRuleRepository) *RiggingDeviceService {
	return &RiggingDeviceService{devices: devices, rules: rules}
}

func (s *RiggingDeviceService) List(page, pageSize int, status, search string) ([]dto.RiggingDeviceResponse, int64, error) {
	items, total, err := s.devices.List(page, pageSize, status, search)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]dto.RiggingDeviceResponse, 0, len(items))
	for _, item := range items {
		response, mapErr := s.withRules(item)
		if mapErr != nil {
			return nil, 0, mapErr
		}
		responses = append(responses, response)
	}
	return responses, total, nil
}

func (s *RiggingDeviceService) Get(id uint) (dto.RiggingDeviceResponse, error) {
	item, err := s.devices.Get(id)
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.withRules(item)
}

func (s *RiggingDeviceService) Create(request dto.CreateRiggingDeviceRequest, actor audit.ActorContext) (dto.RiggingDeviceResponse, error) {
	if err := validateDeviceEnvelope(request.TravelMinM, request.TravelMaxM); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	item := model.RiggingDevice{DeviceCode: normalizeCode(request.DeviceCode), Name: strings.TrimSpace(request.Name), DeviceType: request.DeviceType, MaxLoadKG: request.MaxLoadKG, MaxSpeedMS: request.MaxSpeedMS, TravelMinM: request.TravelMinM, TravelMaxM: request.TravelMaxM, SafetyZone: strings.ToLower(strings.TrimSpace(request.SafetyZone)), DeviceStatus: request.DeviceStatus, Version: 1}
	after := util.SummaryJSON(map[string]any{"device_code": item.DeviceCode, "limits": map[string]any{"max_load_kg": item.MaxLoadKG, "max_speed_ms": item.MaxSpeedMS, "travel_min_m": item.TravelMinM, "travel_max_m": item.TravelMaxM}, "safety_zone": item.SafetyZone, "status": item.DeviceStatus, "version": item.Version})
	if err := s.devices.Create(&item, audit.NewEvent(actor, "rigging_device.create", "rigging_device", 0, "{}", after)); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.withRules(item)
}

func (s *RiggingDeviceService) Update(id uint, request dto.UpdateRiggingDeviceRequest, actor audit.ActorContext) (dto.RiggingDeviceResponse, error) {
	if err := validateDeviceEnvelope(request.TravelMinM, request.TravelMaxM); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	current, err := s.devices.Get(id)
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	before := util.SummaryJSON(map[string]any{"limits": map[string]any{"max_load_kg": current.MaxLoadKG, "max_speed_ms": current.MaxSpeedMS, "travel_min_m": current.TravelMinM, "travel_max_m": current.TravelMaxM}, "safety_zone": current.SafetyZone, "status": current.DeviceStatus, "version": current.Version})
	current.Name = strings.TrimSpace(request.Name)
	current.DeviceType = request.DeviceType
	current.MaxLoadKG = request.MaxLoadKG
	current.MaxSpeedMS = request.MaxSpeedMS
	current.TravelMinM = request.TravelMinM
	current.TravelMaxM = request.TravelMaxM
	current.SafetyZone = strings.ToLower(strings.TrimSpace(request.SafetyZone))
	current.DeviceStatus = request.DeviceStatus
	after := util.SummaryJSON(map[string]any{"limits": map[string]any{"max_load_kg": current.MaxLoadKG, "max_speed_ms": current.MaxSpeedMS, "travel_min_m": current.TravelMinM, "travel_max_m": current.TravelMaxM}, "safety_zone": current.SafetyZone, "status": current.DeviceStatus, "version": request.Version + 1})
	if err := s.devices.Update(&current, request.Version, audit.NewEvent(actor, "rigging_device.update_limits", "rigging_device", id, before, after)); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.withRules(current)
}

func (s *RiggingDeviceService) withRules(item model.RiggingDevice) (dto.RiggingDeviceResponse, error) {
	response := dto.RiggingDeviceFromModel(item)
	rules, _, err := s.rules.List(1, 200, "", "", "")
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	for _, rule := range rules {
		ids := []uint{}
		if err := json.Unmarshal(rule.DeviceIDsJSON, &ids); err != nil {
			return dto.RiggingDeviceResponse{}, fmt.Errorf("decode rule %s device scope: %w", rule.RuleCode, err)
		}
		for _, ruleDeviceID := range ids {
			if ruleDeviceID == item.ID {
				response.ApplicableRules = append(response.ApplicableRules, dto.RuleReference{ID: rule.ID, RuleCode: rule.RuleCode, RuleType: rule.RuleType, Severity: rule.Severity, Enabled: rule.Enabled, RuleVersion: rule.RuleVersion})
				break
			}
		}
	}
	return response, nil
}

func validateDeviceEnvelope(minimum, maximum float64) error {
	if maximum <= minimum {
		return util.Unprocessable("DEVICE_TRAVEL_INVALID", "travel_max_m must be greater than travel_min_m", map[string]any{"travel_min_m": minimum, "travel_max_m": maximum})
	}
	return nil
}

func normalizeCode(value string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", "-"))
}
