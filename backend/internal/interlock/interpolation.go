package interlock

import (
	"fmt"
	"math"
)

func InterpolatePosition(event TimelineEvent, atMS int64) (float64, error) {
	if event.EndMS <= event.StartMS {
		return 0, fmt.Errorf("event %s/%s has a non-positive duration", event.CueCode, event.DeviceCode)
	}
	if atMS < event.StartMS || atMS > event.EndMS {
		return 0, fmt.Errorf("time %d is outside event window %d-%d", atMS, event.StartMS, event.EndMS)
	}
	ratio := float64(atMS-event.StartMS) / float64(event.EndMS-event.StartMS)
	return event.FromPositionM + (event.ToPositionM-event.FromPositionM)*ratio, nil
}

func ActionSpeed(action ActionInput) float64 {
	if action.DurationMS <= 0 {
		return math.Inf(1)
	}
	distance := math.Abs(action.ToPositionM - action.FromPositionM)
	return distance / (float64(action.DurationMS) / 1000)
}

func OverlapWindow(leftStart, leftEnd, rightStart, rightEnd int64) (int64, int64, bool) {
	start := leftStart
	if rightStart > start {
		start = rightStart
	}
	end := leftEnd
	if rightEnd < end {
		end = rightEnd
	}
	return start, end, start < end
}

func MinimumPosition(event TimelineEvent) float64 {
	return math.Min(event.FromPositionM, event.ToPositionM)
}

func MaximumPosition(event TimelineEvent) float64 {
	return math.Max(event.FromPositionM, event.ToPositionM)
}
