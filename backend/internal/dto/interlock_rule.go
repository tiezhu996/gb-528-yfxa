package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"stage-rigging-cue-interlock/backend/internal/interlock"
	"stage-rigging-cue-interlock/backend/internal/model"
)

type CreateInterlockRuleRequest struct {
	RuleCode    string                  `json:"rule_code" binding:"required,min=3,max=48"`
	RuleType    string                  `json:"rule_type" binding:"required,oneof=load_limit speed_limit travel_limit zone_exclusion dependency_guard"`
	DeviceIDs   []uint                  `json:"device_ids" binding:"max=80,dive,gte=1"`
	Threshold   interlock.RuleThreshold `json:"threshold" binding:"required"`
	Severity    string                  `json:"severity" binding:"required,oneof=warning blocker"`
	Enabled     bool                    `json:"enabled"`
	Explanation string                  `json:"explanation" binding:"required,min=8,max=600"`
}

type UpdateInterlockRuleRequest struct {
	DeviceIDs   []uint                  `json:"device_ids" binding:"max=80,dive,gte=1"`
	Threshold   interlock.RuleThreshold `json:"threshold" binding:"required"`
	Severity    string                  `json:"severity" binding:"required,oneof=warning blocker"`
	Enabled     bool                    `json:"enabled"`
	Explanation string                  `json:"explanation" binding:"required,min=8,max=600"`
	RuleVersion uint                    `json:"rule_version" binding:"required,gte=1"`
}

type ToggleRuleRequest struct {
	Enabled     bool `json:"enabled"`
	RuleVersion uint `json:"rule_version" binding:"required,gte=1"`
}

type TestRuleRequest struct {
	ActualValue float64 `json:"actual_value" binding:"gte=-100000,lte=100000"`
}

type TestRuleResponse struct {
	RuleCode       string  `json:"rule_code"`
	RuleType       string  `json:"rule_type"`
	Result         string  `json:"result"`
	ActualValue    float64 `json:"actual_value"`
	ThresholdValue float64 `json:"threshold_value"`
	Unit           string  `json:"unit"`
	Explanation    string  `json:"explanation"`
	Boundary       string  `json:"boundary"`
}

type InterlockRuleResponse struct {
	ID          uint                    `json:"id"`
	RuleCode    string                  `json:"rule_code"`
	RuleType    string                  `json:"rule_type"`
	DeviceIDs   []uint                  `json:"device_ids"`
	Threshold   interlock.RuleThreshold `json:"threshold"`
	Severity    string                  `json:"severity"`
	Enabled     bool                    `json:"enabled"`
	RuleVersion uint                    `json:"rule_version"`
	Explanation string                  `json:"explanation"`
	CreatedAt   time.Time               `json:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at"`
}

func RuleFromModel(item model.InterlockRule) (InterlockRuleResponse, error) {
	deviceIDs := []uint{}
	if err := json.Unmarshal(item.DeviceIDsJSON, &deviceIDs); err != nil {
		return InterlockRuleResponse{}, fmt.Errorf("decode rule %s devices: %w", item.RuleCode, err)
	}
	threshold := interlock.RuleThreshold{}
	if err := json.Unmarshal(item.ThresholdJSON, &threshold); err != nil {
		return InterlockRuleResponse{}, fmt.Errorf("decode rule %s threshold: %w", item.RuleCode, err)
	}
	return InterlockRuleResponse{ID: item.ID, RuleCode: item.RuleCode, RuleType: item.RuleType, DeviceIDs: deviceIDs, Threshold: threshold, Severity: item.Severity, Enabled: item.Enabled, RuleVersion: item.RuleVersion, Explanation: item.Explanation, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}
