package interlock

import "testing"

func TestExpandTimelineAndCollisionWindows(t *testing.T) {
	ordered := []GraphCue{{ID: 1, CueCode: "Q-1", SequenceNo: 1, DurationMS: 1000}, {ID: 2, CueCode: "Q-2", SequenceNo: 2, StartOffsetMS: 500, DurationMS: 1000}}
	cues := map[uint]CueInput{
		1: {GraphCue: ordered[0], Version: 4, Actions: []ActionInput{{DeviceID: 1, DurationMS: 1000, FromPositionM: 10, ToPositionM: 9, LoadKG: 100}}},
		2: {GraphCue: ordered[1], Version: 4, Actions: []ActionInput{{DeviceID: 2, DurationMS: 1000, FromPositionM: 12, ToPositionM: 11, LoadKG: 120}}},
	}
	devices := map[uint]DeviceInput{
		1: {ID: 1, DeviceCode: "D-1", Name: "One", SafetyZone: "zone-a", DeviceStatus: "available"},
		2: {ID: 2, DeviceCode: "D-2", Name: "Two", SafetyZone: "zone-a", DeviceStatus: "available"},
	}
	events, err := ExpandTimeline(ordered, cues, devices)
	if err != nil {
		t.Fatalf("ExpandTimeline returned error: %v", err)
	}
	if len(events) != 2 || events[1].StartMS != 500 || events[1].EndMS != 1500 {
		t.Fatalf("unexpected events: %#v", events)
	}
	windows := DetectCollisionWindows(events)
	if len(windows) != 1 || windows[0].StartMS != 500 || windows[0].EndMS != 1000 {
		t.Fatalf("unexpected collision windows: %#v", windows)
	}
}

func TestTouchingIntervalsDoNotOverlap(t *testing.T) {
	start, end, overlap := OverlapWindow(0, 1000, 1000, 2000)
	if overlap {
		t.Fatalf("touching half-open intervals should not overlap: %d-%d", start, end)
	}
}
