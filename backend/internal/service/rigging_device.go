package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/gorm"
)

// freezeRequiredActions lists the only ways a locked cue reference can be
// cleared before a device enters maintenance or is retired.
var freezeRequiredActions = []string{"archive_locked_cue", "revise_locked_cue_to_new_version"}

const (
	deviceStatusAvailable      = "available"
	deviceStatusInspectionHold = "inspection_hold"
	deviceStatusRetired        = "retired"
)

type RiggingDeviceService struct {
	db      *gorm.DB
	devices *repository.RiggingDeviceRepository
	cues    *repository.CueDefinitionRepository
	rules   *repository.InterlockRuleRepository
}

func NewRiggingDeviceService(db *gorm.DB, devices *repository.RiggingDeviceRepository, cues *repository.CueDefinitionRepository, rules *repository.InterlockRuleRepository) *RiggingDeviceService {
	return &RiggingDeviceService{db: db, devices: devices, cues: cues, rules: rules}
}

func (s *RiggingDeviceService) List(page, pageSize int, status, search string) ([]dto.RiggingDeviceResponse, int64, error) {
	items, total, err := s.devices.List(page, pageSize, status, search)
	if err != nil {
		return nil, 0, err
	}
	lockedCues, err := s.cues.AllLocked()
	if err != nil {
		return nil, 0, err
	}
	enabledRules, err := s.rules.Enabled()
	if err != nil {
		return nil, 0, err
	}
	responses := make([]dto.RiggingDeviceResponse, 0, len(items))
	for _, item := range items {
		response := s.withReferences(dto.RiggingDeviceFromModel(item), item.ID, lockedCues, enabledRules)
		responses = append(responses, response)
	}
	return responses, total, nil
}

func (s *RiggingDeviceService) Get(id uint) (dto.RiggingDeviceResponse, error) {
	item, err := s.devices.Get(id)
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	lockedCues, err := s.cues.AllLocked()
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	enabledRules, _, err := s.rules.List(1, 200, "", "", "")
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.withReferences(dto.RiggingDeviceFromModel(item), id, lockedCues, enabledRules), nil
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
	return s.loadDetail(item)
}

func (s *RiggingDeviceService) Update(id uint, request dto.UpdateRiggingDeviceRequest, actor audit.ActorContext) (dto.RiggingDeviceResponse, error) {
	if err := validateDeviceEnvelope(request.TravelMinM, request.TravelMaxM); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	current, err := s.devices.Get(id)
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	before := deviceSummary(current)
	next := current
	next.Name = strings.TrimSpace(request.Name)
	next.DeviceType = request.DeviceType
	next.MaxLoadKG = request.MaxLoadKG
	next.MaxSpeedMS = request.MaxSpeedMS
	next.TravelMinM = request.TravelMinM
	next.TravelMaxM = request.TravelMaxM
	next.SafetyZone = strings.ToLower(strings.TrimSpace(request.SafetyZone))
	next.DeviceStatus = request.DeviceStatus
	after := util.SummaryJSON(map[string]any{"limits": map[string]any{"max_load_kg": next.MaxLoadKG, "max_speed_ms": next.MaxSpeedMS, "travel_min_m": next.TravelMinM, "travel_max_m": next.TravelMaxM}, "safety_zone": next.SafetyZone, "status": next.DeviceStatus, "version": request.Version + 1})
	event := audit.NewEvent(actor, "rigging_device.update_limits", "rigging_device", id, before, after)

	if enteringMaintenance(current.DeviceStatus, request.DeviceStatus) {
		if err := s.applyMaintenanceFreeze(&next, request.Version, request.DeviceStatus, event); err != nil {
			return dto.RiggingDeviceResponse{}, err
		}
	} else if err := s.devices.Update(&next, request.Version, event); err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.loadDetail(next)
}

// applyMaintenanceFreeze performs the safety freeze: the device row is locked,
// every referencing locked cue and enabled interlock rule is read inside the
// same transaction, and any locked cue rejects the status change before a
// single column is written. A concurrent cue lock either commits first (then
// it shows up in the freeze scan and blocks) or waits on the device row (then
// it fails on the non-available device), so exactly one change can succeed.
func (s *RiggingDeviceService) applyMaintenanceFreeze(item *model.RiggingDevice, expectedVersion uint, targetStatus string, event audit.Event) error {
	return repository.RunInTransaction(s.db, func(tx *gorm.DB) error {
		lockedRows, err := s.devices.WithTx(tx).LockByIDs(tx, []uint{item.ID})
		if err != nil {
			return err
		}
		if len(lockedRows) != 1 {
			return util.NotFound("DEVICE_NOT_FOUND", "rigging device was not found")
		}
		lockedCueModels, err := s.cues.WithTx(tx).AllLocked()
		if err != nil {
			return err
		}
		lockedCues := filterCuesReferencingDevice(lockedCueModels, item.ID)
		enabledRuleModels, err := s.rules.WithTx(tx).Enabled()
		if err != nil {
			return err
		}
		enabledRules := filterRulesReferencingDevice(enabledRuleModels, item.ID)
		if len(lockedCues) > 0 {
			cueCodes := make([]string, 0, len(lockedCues))
			cueRefs := make([]dto.LockedCueReference, 0, len(lockedCues))
			for _, cue := range lockedCues {
				cueCodes = append(cueCodes, cue.CueCode)
				cueRefs = append(cueRefs, dto.LockedCueReference{ID: cue.ID, CueCode: cue.CueCode, SequenceNo: cue.SequenceNo, Version: cue.Version})
			}
			ruleRefs := make([]dto.RuleReference, 0, len(enabledRules))
			for _, rule := range enabledRules {
				ruleRefs = append(ruleRefs, dto.RuleReference{ID: rule.ID, RuleCode: rule.RuleCode, RuleType: rule.RuleType, Severity: rule.Severity, Enabled: rule.Enabled, RuleVersion: rule.RuleVersion})
			}
			return &util.AppError{Status: http.StatusConflict, Code: "DEVICE_MAINTENANCE_BLOCKED", Message: fmt.Sprintf("device cannot enter %s until every referencing locked cue is archived or revised into a new draft version: %s", targetStatus, strings.Join(cueCodes, ", ")), Details: map[string]any{"device_id": item.ID, "device_code": item.DeviceCode, "target_status": targetStatus, "locked_cue_codes": cueCodes, "locked_cues": cueRefs, "enabled_interlock_rules": ruleRefs, "required_actions": freezeRequiredActions}}
		}
		return s.devices.WithTx(tx).UpdateInTransaction(tx, item, expectedVersion, event)
	})
}

func (s *RiggingDeviceService) loadDetail(item model.RiggingDevice) (dto.RiggingDeviceResponse, error) {
	lockedCues, err := s.cues.AllLocked()
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	rules, _, err := s.rules.List(1, 200, "", "", "")
	if err != nil {
		return dto.RiggingDeviceResponse{}, err
	}
	return s.withReferences(dto.RiggingDeviceFromModel(item), item.ID, lockedCues, rules), nil
}

// withReferences fills the applicable rule index and the pre-maintenance
// freeze review shown on the device page.
func (s *RiggingDeviceService) withReferences(response dto.RiggingDeviceResponse, deviceID uint, lockedCues []model.CueDefinition, rules []model.InterlockRule) dto.RiggingDeviceResponse {
	for _, rule := range rules {
		ids := []uint{}
		if err := json.Unmarshal(rule.DeviceIDsJSON, &ids); err != nil {
			return response
		}
		for _, ruleDeviceID := range ids {
			if ruleDeviceID == deviceID {
				response.ApplicableRules = append(response.ApplicableRules, dto.RuleReference{ID: rule.ID, RuleCode: rule.RuleCode, RuleType: rule.RuleType, Severity: rule.Severity, Enabled: rule.Enabled, RuleVersion: rule.RuleVersion})
				break
			}
		}
	}
	blockingCues := filterCuesReferencingDevice(lockedCues, deviceID)
	freeze := dto.MaintenanceFreeze{TargetStatus: deviceStatusInspectionHold, LockedCues: []dto.LockedCueReference{}, EnabledInterlockRules: []dto.RuleReference{}, RequiredActions: freezeRequiredActions}
	for _, cue := range blockingCues {
		freeze.LockedCues = append(freeze.LockedCues, dto.LockedCueReference{ID: cue.ID, CueCode: cue.CueCode, SequenceNo: cue.SequenceNo, Version: cue.Version})
	}
	for _, rule := range filterRulesReferencingDevice(rules, deviceID) {
		if rule.Enabled {
			freeze.EnabledInterlockRules = append(freeze.EnabledInterlockRules, dto.RuleReference{ID: rule.ID, RuleCode: rule.RuleCode, RuleType: rule.RuleType, Severity: rule.Severity, Enabled: rule.Enabled, RuleVersion: rule.RuleVersion})
		}
	}
	freeze.Blocked = len(freeze.LockedCues) > 0
	response.MaintenanceFreeze = freeze
	return response
}

// filterCuesReferencingDevice keeps locked cues whose action snapshot uses the
// given device. Decoding the snapshot works identically on PostgreSQL jsonb
// and SQLite and never mutates the immutable locked version.
func filterCuesReferencingDevice(cues []model.CueDefinition, deviceID uint) []model.CueDefinition {
	result := make([]model.CueDefinition, 0)
	for _, cue := range cues {
		actions := []dto.CueAction{}
		if err := json.Unmarshal(cue.ActionsJSON, &actions); err != nil {
			continue
		}
		for _, action := range actions {
			if action.DeviceID == deviceID {
				result = append(result, cue)
				break
			}
		}
	}
	return result
}

func filterRulesReferencingDevice(rules []model.InterlockRule, deviceID uint) []model.InterlockRule {
	result := make([]model.InterlockRule, 0)
	for _, rule := range rules {
		ids := []uint{}
		if err := json.Unmarshal(rule.DeviceIDsJSON, &ids); err != nil {
			continue
		}
		for _, id := range ids {
			if id == deviceID {
				result = append(result, rule)
				break
			}
		}
	}
	return result
}

// enteringMaintenance reports whether an update moves the device out of
// service into the maintenance hold or the retired terminal state. Updates
// that merely revise limits never trigger the safety freeze.
func enteringMaintenance(currentStatus, targetStatus string) bool {
	if targetStatus != deviceStatusInspectionHold && targetStatus != deviceStatusRetired {
		return false
	}
	return currentStatus != targetStatus
}

func deviceSummary(item model.RiggingDevice) string {
	return util.SummaryJSON(map[string]any{"limits": map[string]any{"max_load_kg": item.MaxLoadKG, "max_speed_ms": item.MaxSpeedMS, "travel_min_m": item.TravelMinM, "travel_max_m": item.TravelMaxM}, "safety_zone": item.SafetyZone, "status": item.DeviceStatus, "version": item.Version})
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
