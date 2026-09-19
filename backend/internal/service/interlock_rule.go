package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/interlock"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/datatypes"
)

type InterlockRuleService struct {
	rules   *repository.InterlockRuleRepository
	devices *repository.RiggingDeviceRepository
}

func NewInterlockRuleService(rules *repository.InterlockRuleRepository, devices *repository.RiggingDeviceRepository) *InterlockRuleService {
	return &InterlockRuleService{rules: rules, devices: devices}
}

func (s *InterlockRuleService) List(page, pageSize int, ruleType, severity, search string) ([]dto.InterlockRuleResponse, int64, error) {
	items, total, err := s.rules.List(page, pageSize, ruleType, severity, search)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]dto.InterlockRuleResponse, 0, len(items))
	for _, item := range items {
		response, mapErr := dto.RuleFromModel(item)
		if mapErr != nil {
			return nil, 0, mapErr
		}
		responses = append(responses, response)
	}
	return responses, total, nil
}

func (s *InterlockRuleService) Get(id uint) (dto.InterlockRuleResponse, error) {
	item, err := s.rules.Get(id)
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	return dto.RuleFromModel(item)
}

func (s *InterlockRuleService) Create(request dto.CreateInterlockRuleRequest, actor audit.ActorContext) (dto.InterlockRuleResponse, error) {
	if err := s.validate(request.RuleType, request.DeviceIDs, request.Threshold); err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	deviceJSON, thresholdJSON, err := marshalRuleInput(request.DeviceIDs, request.Threshold)
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	item := model.InterlockRule{RuleCode: normalizeCode(request.RuleCode), RuleType: request.RuleType, DeviceIDsJSON: deviceJSON, ThresholdJSON: thresholdJSON, Severity: request.Severity, Enabled: request.Enabled, RuleVersion: 1, Explanation: strings.TrimSpace(request.Explanation)}
	after := util.SummaryJSON(map[string]any{"rule_code": item.RuleCode, "rule_type": item.RuleType, "device_ids": request.DeviceIDs, "threshold": request.Threshold, "severity": item.Severity, "enabled": item.Enabled, "rule_version": item.RuleVersion})
	if err := s.rules.Create(&item, audit.NewEvent(actor, "interlock_rule.create", "interlock_rule", 0, "{}", after)); err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	return dto.RuleFromModel(item)
}

func (s *InterlockRuleService) Update(id uint, request dto.UpdateInterlockRuleRequest, actor audit.ActorContext) (dto.InterlockRuleResponse, error) {
	current, err := s.rules.Get(id)
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	if err := s.validate(current.RuleType, request.DeviceIDs, request.Threshold); err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	before, err := dto.RuleFromModel(current)
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	deviceJSON, thresholdJSON, err := marshalRuleInput(request.DeviceIDs, request.Threshold)
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	current.DeviceIDsJSON = deviceJSON
	current.ThresholdJSON = thresholdJSON
	current.Severity = request.Severity
	current.Explanation = strings.TrimSpace(request.Explanation)
	after := util.SummaryJSON(map[string]any{"device_ids": request.DeviceIDs, "threshold": request.Threshold, "severity": request.Severity, "enabled": current.Enabled, "rule_version": request.RuleVersion + 1})
	if err := s.rules.Update(&current, request.RuleVersion, audit.NewEvent(actor, "interlock_rule.update", "interlock_rule", id, util.SummaryJSON(before), after)); err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	return dto.RuleFromModel(current)
}

func (s *InterlockRuleService) Toggle(id uint, request dto.ToggleRuleRequest, actor audit.ActorContext) (dto.InterlockRuleResponse, error) {
	current, err := s.rules.Get(id)
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	before := util.SummaryJSON(map[string]any{"enabled": current.Enabled, "rule_version": current.RuleVersion})
	after := util.SummaryJSON(map[string]any{"enabled": request.Enabled, "rule_version": request.RuleVersion + 1})
	updated, err := s.rules.Toggle(id, request.RuleVersion, request.Enabled, audit.NewEvent(actor, "interlock_rule.toggle", "interlock_rule", id, before, after))
	if err != nil {
		return dto.InterlockRuleResponse{}, err
	}
	return dto.RuleFromModel(updated)
}

func (s *InterlockRuleService) Test(id uint, actual float64) (dto.TestRuleResponse, error) {
	item, err := s.rules.Get(id)
	if err != nil {
		return dto.TestRuleResponse{}, err
	}
	rule, err := dto.RuleFromModel(item)
	if err != nil {
		return dto.TestRuleResponse{}, err
	}
	threshold, unit, pass, err := thresholdComparison(rule.RuleType, rule.Threshold, actual)
	if err != nil {
		return dto.TestRuleResponse{}, err
	}
	result := constants.ResultPass
	if !pass {
		result = constants.InterlockResult(rule.Severity)
	}
	return dto.TestRuleResponse{RuleCode: rule.RuleCode, RuleType: rule.RuleType, Result: string(result), ActualValue: actual, ThresholdValue: threshold, Unit: unit, Explanation: rule.Explanation, Boundary: "Threshold test is offline evidence only; it does not command or authorize stage machinery."}, nil
}

func (s *InterlockRuleService) validate(ruleType string, deviceIDs []uint, threshold interlock.RuleThreshold) error {
	if ruleType != "dependency_guard" && len(deviceIDs) == 0 {
		return util.Unprocessable("RULE_DEVICE_SCOPE_REQUIRED", "this rule type requires at least one device", nil)
	}
	if len(deviceIDs) > 0 {
		if _, err := s.devices.ByIDs(deviceIDs); err != nil {
			return err
		}
	}
	switch ruleType {
	case "load_limit":
		if !threshold.UseDeviceLimits && (threshold.MaxLoadKG == nil || *threshold.MaxLoadKG <= 0) {
			return util.Unprocessable("RULE_THRESHOLD_INVALID", "load_limit requires max_load_kg or use_device_limits", nil)
		}
	case "speed_limit":
		if !threshold.UseDeviceLimits && (threshold.MaxSpeedMS == nil || *threshold.MaxSpeedMS <= 0) {
			return util.Unprocessable("RULE_THRESHOLD_INVALID", "speed_limit requires max_speed_ms or use_device_limits", nil)
		}
	case "travel_limit":
		if !threshold.UseDeviceLimits && (threshold.MinPositionM == nil || threshold.MaxPositionM == nil || *threshold.MaxPositionM <= *threshold.MinPositionM) {
			return util.Unprocessable("RULE_THRESHOLD_INVALID", "travel_limit requires an ordered min/max envelope or use_device_limits", nil)
		}
	case "zone_exclusion", "dependency_guard":
		if threshold.MinimumGapMS != nil && *threshold.MinimumGapMS < 0 {
			return util.Unprocessable("RULE_THRESHOLD_INVALID", "minimum_gap_ms cannot be negative", nil)
		}
	default:
		return util.Unprocessable("RULE_TYPE_INVALID", "unsupported interlock rule type", map[string]any{"rule_type": ruleType})
	}
	return nil
}

func thresholdComparison(ruleType string, threshold interlock.RuleThreshold, actual float64) (float64, string, bool, error) {
	switch ruleType {
	case "load_limit":
		if threshold.MaxLoadKG == nil {
			return 0, "kg", false, util.Unprocessable("RULE_TEST_REQUIRES_DEVICE", "device-limit rules must be tested by a rehearsal action", nil)
		}
		return *threshold.MaxLoadKG, "kg", actual <= *threshold.MaxLoadKG, nil
	case "speed_limit":
		if threshold.MaxSpeedMS == nil {
			return 0, "m/s", false, util.Unprocessable("RULE_TEST_REQUIRES_DEVICE", "device-limit rules must be tested by a rehearsal action", nil)
		}
		return *threshold.MaxSpeedMS, "m/s", actual <= *threshold.MaxSpeedMS, nil
	case "zone_exclusion":
		return 0, "ms overlap", math.Abs(actual) == 0, nil
	case "dependency_guard":
		minimum := float64(0)
		if threshold.MinimumGapMS != nil {
			minimum = *threshold.MinimumGapMS
		}
		return minimum, "ms gap", actual >= minimum, nil
	default:
		return 0, "", false, util.Unprocessable("RULE_TEST_REQUIRES_TIMELINE", "travel rules require modeled endpoints and cannot be reduced to one value", nil)
	}
}

func marshalRuleInput(deviceIDs []uint, threshold interlock.RuleThreshold) (datatypes.JSON, datatypes.JSON, error) {
	deviceJSON, err := json.Marshal(deviceIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("encode rule device scope: %w", err)
	}
	thresholdJSON, err := json.Marshal(threshold)
	if err != nil {
		return nil, nil, fmt.Errorf("encode rule threshold: %w", err)
	}
	return datatypes.JSON(deviceJSON), datatypes.JSON(thresholdJSON), nil
}
