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

	"gorm.io/gorm"
)

type RiggingDeviceService struct {
	devices *repository.RiggingDeviceRepository
	cues    *repository.CueDefinitionRepository
	rules   *repository.InterlockRuleRepository
}

func NewRiggingDeviceService(devices *repository.RiggingDeviceRepository, cues *repository.CueDefinitionRepository, rules *repository.InterlockRuleRepository) *RiggingDeviceService {
	return &RiggingDeviceService{devices: devices, cues: cues, rules: rules}
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
	// Maintenance freeze: moving into inspection_hold or retired is rejected
	// inside the update transaction while locked cues still reference the
	// device, so the status can never change first and be reconciled later.
	var guard repository.Guard
	if freezesDevice(current.DeviceStatus, request.DeviceStatus) {
		guard = func(tx *gorm.DB) error {
			report, reportErr := s.freezeReport(s.cues.WithTx(tx), s.rules.WithTx(tx), current)
			if reportErr != nil {
				return reportErr
			}
			if report.Blocked {
				return util.ConflictDetails("DEVICE_MAINTENANCE_BLOCKED", "locked cues still reference this device; archive each cue or create a new cue version before maintenance", report)
			}
			return nil
		}
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
	if err := s.devices.UpdateGuarded(&current, request.Version, guard, audit.NewEvent(actor, "rigging_device.update_limits", "rigging_device", id, before, after)); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.withRules(current)
}

// MaintenanceCheck lists the locked cues and enabled interlock rules that
// still reference the device, without changing any state.
func (s *RiggingDeviceService) MaintenanceCheck(id uint) (dto.DeviceFreezeReport, error) {
	device, err := s.devices.Get(id)
	if err != nil {
		return dto.DeviceFreezeReport{}, err
	}
	return s.freezeReport(s.cues, s.rules, device)
}

func (s *RiggingDeviceService) freezeReport(cues *repository.CueDefinitionRepository, rules *repository.InterlockRuleRepository, device model.RiggingDevice) (dto.DeviceFreezeReport, error) {
	report := dto.DeviceFreezeReport{DeviceID: device.ID, DeviceCode: device.DeviceCode, LockedCues: []dto.CueReference{}, EnabledRules: []dto.RuleReference{}}
	locked, err := cues.ListLocked()
	if err != nil {
		return dto.DeviceFreezeReport{}, err
	}
	for _, cue := range locked {
		actions := []dto.CueAction{}
		if err := json.Unmarshal(cue.ActionsJSON, &actions); err != nil {
			return dto.DeviceFreezeReport{}, fmt.Errorf("decode cue %s actions: %w", cue.CueCode, err)
		}
		for _, action := range actions {
			if action.DeviceID == device.ID {
				report.LockedCues = append(report.LockedCues, dto.CueReference{ID: cue.ID, CueCode: cue.CueCode, SequenceNo: cue.SequenceNo, Version: cue.Version})
				break
			}
		}
	}
	enabled, err := rules.Enabled()
	if err != nil {
		return dto.DeviceFreezeReport{}, err
	}
	references, err := applicableRules(enabled, device.ID)
	if err != nil {
		return dto.DeviceFreezeReport{}, err
	}
	report.EnabledRules = references
	report.Blocked = len(report.LockedCues) > 0
	return report, nil
}

func freezesDevice(from, to string) bool {
	return from != to && (to == model.DeviceStatusInspectionHold || to == model.DeviceStatusRetired)
}

func (s *RiggingDeviceService) withRules(item model.RiggingDevice) (dto.RiggingDeviceResponse, error) {
	response := dto.RiggingDeviceFromModel(item)
	rules, _, err := s.rules.List(1, 200, "", "", "")
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	references, err := applicableRules(rules, item.ID)
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	response.ApplicableRules = references
	return response, nil
}

func applicableRules(rules []model.InterlockRule, deviceID uint) ([]dto.RuleReference, error) {
	references := []dto.RuleReference{}
	for _, rule := range rules {
		ids := []uint{}
		if err := json.Unmarshal(rule.DeviceIDsJSON, &ids); err != nil {
			return nil, fmt.Errorf("decode rule %s device scope: %w", rule.RuleCode, err)
		}
		for _, ruleDeviceID := range ids {
			if ruleDeviceID == deviceID {
				references = append(references, dto.RuleReference{ID: rule.ID, RuleCode: rule.RuleCode, RuleType: rule.RuleType, Severity: rule.Severity, Enabled: rule.Enabled, RuleVersion: rule.RuleVersion})
				break
			}
		}
	}
	return references, nil
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
