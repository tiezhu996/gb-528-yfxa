package interlock

import (
	"math"
	"testing"
)

func TestInterpolatePosition(t *testing.T) {
	event := TimelineEvent{CueCode: "Q-1", DeviceCode: "D-1", StartMS: 1000, EndMS: 5000, FromPositionM: 12, ToPositionM: 4}
	tests := []struct {
		at   int64
		want float64
	}{
		{1000, 12},
		{3000, 8},
		{5000, 4},
	}
	for _, test := range tests {
		got, err := InterpolatePosition(event, test.at)
		if err != nil {
			t.Fatalf("InterpolatePosition(%d) returned error: %v", test.at, err)
		}
		if math.Abs(got-test.want) > 1e-9 {
			t.Fatalf("InterpolatePosition(%d) = %f, want %f", test.at, got, test.want)
		}
	}
	if _, err := InterpolatePosition(event, 999); err == nil {
		t.Fatal("expected an out-of-window error")
	}
}

func TestActionSpeed(t *testing.T) {
	got := ActionSpeed(ActionInput{DurationMS: 2000, FromPositionM: 10, ToPositionM: 8})
	if math.Abs(got-1) > 1e-9 {
		t.Fatalf("ActionSpeed = %f, want 1", got)
	}
}
