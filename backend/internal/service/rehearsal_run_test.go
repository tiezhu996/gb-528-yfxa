package service

import (
	"testing"

	"stage-rigging-cue-interlock/backend/internal/model"
)

func TestCueSetVersionIncludesDeviceVersion(t *testing.T) {
	devices := []model.RiggingDevice{{ID: 7, Version: 1}}
	first := cueSetVersion(nil, nil, devices)
	devices[0].Version = 2
	second := cueSetVersion(nil, nil, devices)
	if first == second {
		t.Fatal("device version change must create a distinct rehearsal input version")
	}
}
