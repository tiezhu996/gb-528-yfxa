package interlock

import (
	"fmt"
	"sort"

	"stage-rigging-cue-interlock/backend/internal/constants"
)

type RuleEvidence struct {
	RuleCode       string                    `json:"rule_code"`
	RuleType       string                    `json:"rule_type"`
	Result         constants.InterlockResult `json:"result"`
	Severity       string                    `json:"severity"`
	CueCodes       []string                  `json:"cue_codes"`
	DeviceCodes    []string                  `json:"device_codes"`
	WindowStartMS  int64                     `json:"window_start_ms"`
	WindowEndMS    int64                     `json:"window_end_ms"`
	ActualValue    float64                   `json:"actual_value"`
	ThresholdValue float64                   `json:"threshold_value"`
	Unit           string                    `json:"unit"`
	Message        string                    `json:"message"`
}

type Evaluation struct {
	OrderedCues      []GraphCue                `json:"ordered_cues"`
	Timeline         []TimelineEvent           `json:"timeline"`
	RuleResults      []RuleEvidence            `json:"rule_results"`
	CollisionWindows []CollisionWindow         `json:"collision_windows"`
	HighestSeverity  constants.InterlockResult `json:"highest_severity"`
	TimelineStepMS   int64                     `json:"timeline_step_ms"`
	Assumptions      []string                  `json:"assumptions"`
}

func Evaluate(cueInputs []CueInput, deviceInputs []DeviceInput, rules []RuleInput, timelineStepMS int64) (Evaluation, error) {
	graphCues := make([]GraphCue, 0, len(cueInputs))
	cues := make(map[uint]CueInput, len(cueInputs))
	for _, cue := range cueInputs {
		graphCues = append(graphCues, cue.GraphCue)
		cues[cue.ID] = cue
	}
	ordered, err := ValidateAndSort(graphCues)
	if err != nil {
		return Evaluation{}, err
	}
	devices := make(map[uint]DeviceInput, len(deviceInputs))
	for _, device := range deviceInputs {
		devices[device.ID] = device
	}
	timeline, err := ExpandTimeline(ordered, cues, devices)
	if err != nil {
		return Evaluation{}, err
	}
	windows := DetectCollisionWindows(timeline)
	evidence := make([]RuleEvidence, 0)
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		results, evalErr := evaluateRule(rule, ordered, cues, devices, timeline, windows)
		if evalErr != nil {
			return Evaluation{}, fmt.Errorf("evaluate rule %s: %w", rule.RuleCode, evalErr)
		}
		evidence = append(evidence, results...)
	}
	if len(evidence) == 0 {
		evidence = append(evidence, RuleEvidence{RuleCode: "RULESET", RuleType: "summary", Result: constants.ResultPass, Severity: "informational", Message: "No enabled rule produced a warning or blocker for this snapshot."})
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		if constants.SeverityRank(evidence[i].Result) == constants.SeverityRank(evidence[j].Result) {
			if evidence[i].RuleCode == evidence[j].RuleCode {
				return evidence[i].WindowStartMS < evidence[j].WindowStartMS
			}
			return evidence[i].RuleCode < evidence[j].RuleCode
		}
		return constants.SeverityRank(evidence[i].Result) > constants.SeverityRank(evidence[j].Result)
	})
	severity := constants.ResultPass
	for _, result := range evidence {
		severity = constants.HighestSeverity(severity, result.Result)
	}
	return Evaluation{OrderedCues: ordered, Timeline: timeline, RuleResults: evidence, CollisionWindows: windows, HighestSeverity: severity, TimelineStepMS: timelineStepMS, Assumptions: []string{
		"Positions are linearly interpolated between each action's modeled endpoints.",
		"Intervals are half-open: an action ending exactly when another begins does not overlap.",
		"Rules and device limits are evaluated from the immutable snapshot stored with the run.",
		"Results are offline rehearsal evidence only and never authorize or command machinery.",
	}}, nil
}

func evaluateRule(rule RuleInput, ordered []GraphCue, cues map[uint]CueInput, devices map[uint]DeviceInput, timeline []TimelineEvent, windows []CollisionWindow) ([]RuleEvidence, error) {
	selected := make(map[uint]bool, len(rule.DeviceIDs))
	for _, id := range rule.DeviceIDs {
		if _, exists := devices[id]; !exists {
			return nil, fmt.Errorf("references missing device id %d", id)
		}
		selected[id] = true
	}
	switch rule.RuleType {
	case "load_limit":
		return evaluateLoad(rule, selected, devices, timeline), nil
	case "speed_limit":
		return evaluateSpeed(rule, selected, devices, timeline), nil
	case "travel_limit":
		return evaluateTravel(rule, selected, devices, timeline), nil
	case "zone_exclusion":
		return evaluateZones(rule, selected, windows), nil
	case "dependency_guard":
		return evaluateDependencies(rule, ordered, cues), nil
	default:
		return nil, fmt.Errorf("unsupported rule type %q", rule.RuleType)
	}
}

func evaluateLoad(rule RuleInput, selected map[uint]bool, devices map[uint]DeviceInput, events []TimelineEvent) []RuleEvidence {
	results := make([]RuleEvidence, 0)
	for _, event := range events {
		if len(selected) > 0 && !selected[event.DeviceID] {
			continue
		}
		threshold := devices[event.DeviceID].MaxLoadKG
		if rule.Threshold.MaxLoadKG != nil && !rule.Threshold.UseDeviceLimits {
			threshold = *rule.Threshold.MaxLoadKG
		}
		if event.LoadKG > threshold {
			results = append(results, breach(rule, event, event.LoadKG, threshold, "kg", "modeled load exceeds the rule threshold"))
		}
	}
	return withPass(rule, results, "All selected modeled loads are within the rule threshold.")
}

func evaluateSpeed(rule RuleInput, selected map[uint]bool, devices map[uint]DeviceInput, events []TimelineEvent) []RuleEvidence {
	results := make([]RuleEvidence, 0)
	for _, event := range events {
		if len(selected) > 0 && !selected[event.DeviceID] {
			continue
		}
		threshold := devices[event.DeviceID].MaxSpeedMS
		if rule.Threshold.MaxSpeedMS != nil && !rule.Threshold.UseDeviceLimits {
			threshold = *rule.Threshold.MaxSpeedMS
		}
		if event.SpeedMS > threshold {
			results = append(results, breach(rule, event, event.SpeedMS, threshold, "m/s", "modeled linear speed exceeds the rule threshold"))
		}
	}
	return withPass(rule, results, "All selected modeled speeds are within the rule threshold.")
}

func evaluateTravel(rule RuleInput, selected map[uint]bool, devices map[uint]DeviceInput, events []TimelineEvent) []RuleEvidence {
	results := make([]RuleEvidence, 0)
	for _, event := range events {
		if len(selected) > 0 && !selected[event.DeviceID] {
			continue
		}
		device := devices[event.DeviceID]
		minimum, maximum := device.TravelMinM, device.TravelMaxM
		if rule.Threshold.MinPositionM != nil && !rule.Threshold.UseDeviceLimits {
			minimum = *rule.Threshold.MinPositionM
		}
		if rule.Threshold.MaxPositionM != nil && !rule.Threshold.UseDeviceLimits {
			maximum = *rule.Threshold.MaxPositionM
		}
		if MinimumPosition(event) < minimum {
			results = append(results, breach(rule, event, MinimumPosition(event), minimum, "m", "modeled endpoint is below the travel minimum"))
		}
		if MaximumPosition(event) > maximum {
			results = append(results, breach(rule, event, MaximumPosition(event), maximum, "m", "modeled endpoint is above the travel maximum"))
		}
	}
	return withPass(rule, results, "All selected modeled endpoints are within their travel envelopes.")
}

func evaluateZones(rule RuleInput, selected map[uint]bool, windows []CollisionWindow) []RuleEvidence {
	results := make([]RuleEvidence, 0)
	for _, window := range windows {
		if len(selected) > 0 {
			matched := 0
			for _, deviceID := range window.DeviceIDs {
				if selected[deviceID] {
					matched++
				}
			}
			if matched < 2 {
				continue
			}
		}
		results = append(results, RuleEvidence{RuleCode: rule.RuleCode, RuleType: rule.RuleType, Result: severityResult(rule.Severity), Severity: rule.Severity, CueCodes: window.CueCodes, DeviceCodes: window.DeviceCodes, WindowStartMS: window.StartMS, WindowEndMS: window.EndMS, ActualValue: float64(window.EndMS - window.StartMS), ThresholdValue: 0, Unit: "ms overlap", Message: "modeled actions overlap inside safety zone " + window.SafetyZone})
	}
	return withPass(rule, results, "No selected devices have overlapping modeled motion in a shared safety zone.")
}

func evaluateDependencies(rule RuleInput, ordered []GraphCue, cues map[uint]CueInput) []RuleEvidence {
	minimumGap := float64(0)
	if rule.Threshold.MinimumGapMS != nil {
		minimumGap = *rule.Threshold.MinimumGapMS
	}
	results := make([]RuleEvidence, 0)
	for _, cue := range ordered {
		for _, dependencyID := range cue.DependencyIDs {
			dependency := cues[dependencyID]
			gap := float64(cue.StartOffsetMS - (dependency.StartOffsetMS + dependency.DurationMS))
			if gap < minimumGap {
				results = append(results, RuleEvidence{RuleCode: rule.RuleCode, RuleType: rule.RuleType, Result: severityResult(rule.Severity), Severity: rule.Severity, CueCodes: []string{dependency.CueCode, cue.CueCode}, WindowStartMS: dependency.StartOffsetMS + dependency.DurationMS, WindowEndMS: cue.StartOffsetMS, ActualValue: gap, ThresholdValue: minimumGap, Unit: "ms gap", Message: "dependent cue begins before the required predecessor gap is satisfied"})
			}
		}
	}
	return withPass(rule, results, "Every selected cue begins after its declared dependencies and required gap.")
}

func breach(rule RuleInput, event TimelineEvent, actual, threshold float64, unit, message string) RuleEvidence {
	return RuleEvidence{RuleCode: rule.RuleCode, RuleType: rule.RuleType, Result: severityResult(rule.Severity), Severity: rule.Severity, CueCodes: []string{event.CueCode}, DeviceCodes: []string{event.DeviceCode}, WindowStartMS: event.StartMS, WindowEndMS: event.EndMS, ActualValue: actual, ThresholdValue: threshold, Unit: unit, Message: message}
}

func severityResult(severity string) constants.InterlockResult {
	if severity == "warning" {
		return constants.ResultWarning
	}
	return constants.ResultBlocker
}

func withPass(rule RuleInput, results []RuleEvidence, message string) []RuleEvidence {
	if len(results) > 0 {
		return results
	}
	return []RuleEvidence{{RuleCode: rule.RuleCode, RuleType: rule.RuleType, Result: constants.ResultPass, Severity: rule.Severity, Message: message}}
}
