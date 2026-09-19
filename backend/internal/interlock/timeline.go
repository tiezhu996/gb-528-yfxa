package interlock

import (
	"fmt"
	"sort"
)

type ActionInput struct {
	DeviceID      uint    `json:"device_id"`
	StartOffsetMS int64   `json:"start_offset_ms"`
	DurationMS    int64   `json:"duration_ms"`
	FromPositionM float64 `json:"from_position_m"`
	ToPositionM   float64 `json:"to_position_m"`
	LoadKG        float64 `json:"load_kg"`
}

type CueInput struct {
	GraphCue
	Version uint          `json:"version"`
	Actions []ActionInput `json:"actions"`
}

type DeviceInput struct {
	ID           uint    `json:"id"`
	DeviceCode   string  `json:"device_code"`
	Name         string  `json:"name"`
	MaxLoadKG    float64 `json:"max_load_kg"`
	MaxSpeedMS   float64 `json:"max_speed_ms"`
	TravelMinM   float64 `json:"travel_min_m"`
	TravelMaxM   float64 `json:"travel_max_m"`
	SafetyZone   string  `json:"safety_zone"`
	DeviceStatus string  `json:"device_status"`
}

type RuleThreshold struct {
	MaxLoadKG       *float64 `json:"max_load_kg,omitempty"`
	MaxSpeedMS      *float64 `json:"max_speed_ms,omitempty"`
	MinPositionM    *float64 `json:"min_position_m,omitempty"`
	MaxPositionM    *float64 `json:"max_position_m,omitempty"`
	MinimumGapMS    *float64 `json:"minimum_gap_ms,omitempty"`
	UseDeviceLimits bool     `json:"use_device_limits,omitempty"`
}

type RuleInput struct {
	ID          uint          `json:"id"`
	RuleCode    string        `json:"rule_code"`
	RuleType    string        `json:"rule_type"`
	DeviceIDs   []uint        `json:"device_ids"`
	Threshold   RuleThreshold `json:"threshold"`
	Severity    string        `json:"severity"`
	Enabled     bool          `json:"enabled"`
	RuleVersion uint          `json:"rule_version"`
	Explanation string        `json:"explanation"`
}

type TimelineEvent struct {
	CueID         uint    `json:"cue_id"`
	CueCode       string  `json:"cue_code"`
	CueSequence   int     `json:"cue_sequence"`
	CueVersion    uint    `json:"cue_version"`
	DeviceID      uint    `json:"device_id"`
	DeviceCode    string  `json:"device_code"`
	DeviceName    string  `json:"device_name"`
	SafetyZone    string  `json:"safety_zone"`
	StartMS       int64   `json:"start_ms"`
	EndMS         int64   `json:"end_ms"`
	FromPositionM float64 `json:"from_position_m"`
	ToPositionM   float64 `json:"to_position_m"`
	LoadKG        float64 `json:"load_kg"`
	SpeedMS       float64 `json:"speed_ms"`
}

type CollisionWindow struct {
	SafetyZone  string   `json:"safety_zone"`
	CueCodes    []string `json:"cue_codes"`
	DeviceIDs   []uint   `json:"device_ids"`
	DeviceCodes []string `json:"device_codes"`
	StartMS     int64    `json:"start_ms"`
	EndMS       int64    `json:"end_ms"`
}

func ExpandTimeline(ordered []GraphCue, cues map[uint]CueInput, devices map[uint]DeviceInput) ([]TimelineEvent, error) {
	events := make([]TimelineEvent, 0)
	for _, graphCue := range ordered {
		cue, ok := cues[graphCue.ID]
		if !ok {
			return nil, fmt.Errorf("cue %s is missing its action input", graphCue.CueCode)
		}
		if len(cue.Actions) == 0 {
			return nil, fmt.Errorf("cue %s has no actions", graphCue.CueCode)
		}
		seenDevices := map[uint]bool{}
		for _, action := range cue.Actions {
			device, exists := devices[action.DeviceID]
			if !exists {
				return nil, fmt.Errorf("cue %s references missing device id %d", cue.CueCode, action.DeviceID)
			}
			if device.DeviceStatus != "available" {
				return nil, fmt.Errorf("cue %s references device %s in status %s", cue.CueCode, device.DeviceCode, device.DeviceStatus)
			}
			if seenDevices[action.DeviceID] {
				return nil, fmt.Errorf("cue %s contains duplicate action for device %s", cue.CueCode, device.DeviceCode)
			}
			seenDevices[action.DeviceID] = true
			if action.StartOffsetMS < 0 || action.DurationMS <= 0 || action.StartOffsetMS+action.DurationMS > cue.DurationMS {
				return nil, fmt.Errorf("cue %s action for %s exceeds the cue time envelope", cue.CueCode, device.DeviceCode)
			}
			if action.LoadKG < 0 {
				return nil, fmt.Errorf("cue %s action for %s has negative load", cue.CueCode, device.DeviceCode)
			}
			start := cue.StartOffsetMS + action.StartOffsetMS
			end := start + action.DurationMS
			events = append(events, TimelineEvent{CueID: cue.ID, CueCode: cue.CueCode, CueSequence: cue.SequenceNo, CueVersion: cue.Version, DeviceID: device.ID, DeviceCode: device.DeviceCode, DeviceName: device.Name, SafetyZone: device.SafetyZone, StartMS: start, EndMS: end, FromPositionM: action.FromPositionM, ToPositionM: action.ToPositionM, LoadKG: action.LoadKG, SpeedMS: ActionSpeed(action)})
		}
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartMS == events[j].StartMS {
			if events[i].CueSequence == events[j].CueSequence {
				return events[i].DeviceCode < events[j].DeviceCode
			}
			return events[i].CueSequence < events[j].CueSequence
		}
		return events[i].StartMS < events[j].StartMS
	})
	return events, nil
}

func DetectCollisionWindows(events []TimelineEvent) []CollisionWindow {
	windows := make([]CollisionWindow, 0)
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			left, right := events[i], events[j]
			if left.DeviceID == right.DeviceID || left.SafetyZone == "" || left.SafetyZone != right.SafetyZone {
				continue
			}
			start, end, overlaps := OverlapWindow(left.StartMS, left.EndMS, right.StartMS, right.EndMS)
			if overlaps {
				windows = append(windows, CollisionWindow{SafetyZone: left.SafetyZone, CueCodes: []string{left.CueCode, right.CueCode}, DeviceIDs: []uint{left.DeviceID, right.DeviceID}, DeviceCodes: []string{left.DeviceCode, right.DeviceCode}, StartMS: start, EndMS: end})
			}
		}
	}
	return windows
}
