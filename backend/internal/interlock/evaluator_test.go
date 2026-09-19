package interlock

import (
	"reflect"
	"testing"

	"stage-rigging-cue-interlock/backend/internal/constants"
)

func TestEvaluateDeterministicBlockerEvidence(t *testing.T) {
	cues := []CueInput{{GraphCue: GraphCue{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 2000}, Version: 4, Actions: []ActionInput{{DeviceID: 1, DurationMS: 2000, FromPositionM: 10, ToPositionM: 8, LoadKG: 650}}}}
	devices := []DeviceInput{{ID: 1, DeviceCode: "D-1", Name: "One", MaxLoadKG: 600, MaxSpeedMS: 2, TravelMinM: 5, TravelMaxM: 15, SafetyZone: "zone-a", DeviceStatus: "available"}}
	rules := []RuleInput{{ID: 1, RuleCode: "LOAD-1", RuleType: "load_limit", DeviceIDs: []uint{1}, Threshold: RuleThreshold{UseDeviceLimits: true}, Severity: "blocker", Enabled: true, RuleVersion: 1}}
	first, err := Evaluate(cues, devices, rules, 100)
	if err != nil {
		t.Fatalf("first Evaluate returned error: %v", err)
	}
	second, err := Evaluate(cues, devices, rules, 100)
	if err != nil {
		t.Fatalf("second Evaluate returned error: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("fixed input produced different evaluations:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if first.HighestSeverity != constants.ResultBlocker || len(first.RuleResults) != 1 {
		t.Fatalf("unexpected evaluation: %#v", first)
	}
	result := first.RuleResults[0]
	if result.RuleCode != "LOAD-1" || result.ActualValue != 650 || result.ThresholdValue != 600 || len(result.CueCodes) != 1 || len(result.DeviceCodes) != 1 {
		t.Fatalf("incomplete blocker evidence: %#v", result)
	}
}

func TestEvaluateDependencyGap(t *testing.T) {
	zero := float64(0)
	cues := []CueInput{
		{GraphCue: GraphCue{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 1000}, Version: 4, Actions: []ActionInput{{DeviceID: 1, DurationMS: 1000}}},
		{GraphCue: GraphCue{ID: 2, CueCode: "Q-2", SequenceNo: 2, StartOffsetMS: 900, DurationMS: 1000, DependencyIDs: []uint{1}}, Version: 4, Actions: []ActionInput{{DeviceID: 2, DurationMS: 1000}}},
	}
	devices := []DeviceInput{{ID: 1, DeviceCode: "D-1", DeviceStatus: "available", SafetyZone: "a"}, {ID: 2, DeviceCode: "D-2", DeviceStatus: "available", SafetyZone: "b"}}
	rules := []RuleInput{{ID: 1, RuleCode: "DEP-1", RuleType: "dependency_guard", Threshold: RuleThreshold{MinimumGapMS: &zero}, Severity: "blocker", Enabled: true, RuleVersion: 1}}
	evaluation, err := Evaluate(cues, devices, rules, 100)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.HighestSeverity != constants.ResultBlocker || evaluation.RuleResults[0].ActualValue != -100 {
		t.Fatalf("expected -100ms dependency blocker, got %#v", evaluation.RuleResults)
	}
}
