package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var freezeActor = audit.ActorContext{ID: 1, Username: "freeze-tester", RequestID: "req-freeze-test"}

type freezeRig struct {
	db      *gorm.DB
	devices *RiggingDeviceService
	cues    *CueDefinitionService
}

func newFreezeRig(t *testing.T, suffix string) *freezeRig {
	t.Helper()
	name := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-%s?mode=memory&cache=shared", name, suffix)), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if err := db.AutoMigrate(&model.RiggingDevice{}, &model.CueDefinition{}, &model.InterlockRule{}, &audit.Event{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	auditRepository := audit.NewRepository(db)
	deviceRepository := repository.NewRiggingDeviceRepository(db, auditRepository)
	cueRepository := repository.NewCueDefinitionRepository(db, auditRepository)
	ruleRepository := repository.NewInterlockRuleRepository(db, auditRepository)
	return &freezeRig{db: db, devices: NewRiggingDeviceService(deviceRepository, cueRepository, ruleRepository), cues: NewCueDefinitionService(cueRepository, deviceRepository)}
}

func (r *freezeRig) createDevice(t *testing.T, code string) dto.RiggingDeviceResponse {
	t.Helper()
	device, err := r.devices.Create(dto.CreateRiggingDeviceRequest{DeviceCode: code, Name: "Freeze test carrier", DeviceType: "motorized_batten", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 4, TravelMaxM: 16, SafetyZone: "overstage-t", DeviceStatus: model.DeviceStatusAvailable}, freezeActor)
	if err != nil {
		t.Fatalf("create device %s: %v", code, err)
	}
	return device
}

func (r *freezeRig) createCue(t *testing.T, code string, sequence int, deviceID uint) dto.CueDefinitionResponse {
	t.Helper()
	cue, err := r.cues.Create(dto.CreateCueRequest{CueCode: code, Name: "Freeze test cue", SequenceNo: sequence, StartOffsetMS: 0, DurationMS: 8000, Actions: []dto.CueAction{{DeviceID: deviceID, StartOffsetMS: 0, DurationMS: 8000, FromPositionM: 12, ToPositionM: 9, LoadKG: 200}}}, freezeActor)
	if err != nil {
		t.Fatalf("create cue %s: %v", code, err)
	}
	return cue
}

func (r *freezeRig) transitionCue(t *testing.T, cue dto.CueDefinitionResponse, target constants.CueStatus) dto.CueDefinitionResponse {
	t.Helper()
	updated, err := r.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: cue.Version, Reason: "freeze test transition"}, target, freezeActor)
	if err != nil {
		t.Fatalf("transition cue %s to %s: %v", cue.CueCode, target, err)
	}
	return updated
}

func (r *freezeRig) lockCue(t *testing.T, code string, sequence int, deviceID uint) dto.CueDefinitionResponse {
	t.Helper()
	cue := r.createCue(t, code, sequence, deviceID)
	cue = r.transitionCue(t, cue, constants.CuePendingReview)
	cue = r.transitionCue(t, cue, constants.CueApproved)
	return r.transitionCue(t, cue, constants.CueLocked)
}

func (r *freezeRig) updateStatus(device dto.RiggingDeviceResponse, status string) (dto.RiggingDeviceResponse, error) {
	return r.devices.Update(device.ID, dto.UpdateRiggingDeviceRequest{Name: device.Name, DeviceType: device.DeviceType, MaxLoadKG: device.MaxLoadKG, MaxSpeedMS: device.MaxSpeedMS, TravelMinM: device.TravelMinM, TravelMaxM: device.TravelMaxM, SafetyZone: device.SafetyZone, DeviceStatus: status, Version: device.Version}, freezeActor)
}

func TestMaintenanceFreezeRejectsLockedCueReferences(t *testing.T) {
	rig := newFreezeRig(t, "blocked")
	device := rig.createDevice(t, "FRZ-BATTEN-01")
	rig.lockCue(t, "FRZ-Q-010", 10, device.ID)
	rule := model.InterlockRule{RuleCode: "FRZ-LOAD-01", RuleType: "load_limit", DeviceIDsJSON: datatypes.JSON(fmt.Sprintf("[%d]", device.ID)), ThresholdJSON: datatypes.JSON(`{"use_device_limits":true}`), Severity: "blocker", Enabled: true, RuleVersion: 1, Explanation: "freeze test rule"}
	if err := rig.db.Create(&rule).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}

	report, err := rig.devices.MaintenanceCheck(device.ID)
	if err != nil {
		t.Fatalf("maintenance check: %v", err)
	}
	if !report.Blocked || len(report.LockedCues) != 1 || report.LockedCues[0].CueCode != "FRZ-Q-010" {
		t.Fatalf("report must list the locked cue, got %+v", report)
	}
	if len(report.EnabledRules) != 1 || report.EnabledRules[0].RuleCode != "FRZ-LOAD-01" {
		t.Fatalf("report must list the enabled rule, got %+v", report.EnabledRules)
	}

	_, err = rig.updateStatus(device, model.DeviceStatusInspectionHold)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != "DEVICE_MAINTENANCE_BLOCKED" {
		t.Fatalf("expected DEVICE_MAINTENANCE_BLOCKED, got %v", err)
	}
	details, _ := json.Marshal(appErr.Details)
	if !strings.Contains(string(details), "FRZ-Q-010") || !strings.Contains(string(details), "FRZ-LOAD-01") {
		t.Fatalf("rejection must name the blocking cue and enabled rules, got %s", details)
	}

	stored, err := rig.devices.Get(device.ID)
	if err != nil {
		t.Fatalf("reload device: %v", err)
	}
	if stored.DeviceStatus != model.DeviceStatusAvailable || stored.Version != device.Version {
		t.Fatalf("rejected freeze must not leave a half update, got status=%s version=%d", stored.DeviceStatus, stored.Version)
	}
}

func TestMaintenanceFreezeLiftsAfterCueArchived(t *testing.T) {
	rig := newFreezeRig(t, "archive")
	device := rig.createDevice(t, "FRZ-BATTEN-02")
	cue := rig.lockCue(t, "FRZ-Q-020", 20, device.ID)
	rig.transitionCue(t, cue, constants.CueArchived)

	updated, err := rig.updateStatus(device, model.DeviceStatusInspectionHold)
	if err != nil {
		t.Fatalf("maintenance must succeed once the locked cue is archived: %v", err)
	}
	if updated.DeviceStatus != model.DeviceStatusInspectionHold {
		t.Fatalf("expected inspection_hold, got %s", updated.DeviceStatus)
	}
	report, err := rig.devices.MaintenanceCheck(device.ID)
	if err != nil {
		t.Fatalf("maintenance check: %v", err)
	}
	if report.Blocked || len(report.LockedCues) != 0 {
		t.Fatalf("archived cues must not block maintenance, got %+v", report)
	}
}

func TestCueLockRejectedWhileDeviceUnderMaintenance(t *testing.T) {
	rig := newFreezeRig(t, "lock")
	device := rig.createDevice(t, "FRZ-BATTEN-03")
	if _, err := rig.updateStatus(device, model.DeviceStatusInspectionHold); err != nil {
		t.Fatalf("maintenance without locked cues must succeed: %v", err)
	}
	cue := rig.createCue(t, "FRZ-Q-030", 30, device.ID)
	cue = rig.transitionCue(t, cue, constants.CuePendingReview)
	cue = rig.transitionCue(t, cue, constants.CueApproved)

	_, err := rig.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: cue.Version, Reason: "freeze test lock"}, constants.CueLocked, freezeActor)
	var appErr *util.AppError
	if !errors.As(err, &appErr) || appErr.Code != "DEVICE_UNDER_MAINTENANCE" {
		t.Fatalf("expected DEVICE_UNDER_MAINTENANCE, got %v", err)
	}
	stored, err := rig.cues.Get(cue.ID)
	if err != nil {
		t.Fatalf("reload cue: %v", err)
	}
	if stored.CueStatus != constants.CueApproved {
		t.Fatalf("failed lock must roll the cue back to approved, got %s", stored.CueStatus)
	}
}

func TestDeviceMaintenanceAndCueLockAreMutuallyExclusive(t *testing.T) {
	for attempt := 0; attempt < 6; attempt++ {
		rig := newFreezeRig(t, fmt.Sprintf("race-%d", attempt))
		device := rig.createDevice(t, "FRZ-RACE-01")
		cue := rig.createCue(t, "FRZ-Q-500", 500, device.ID)
		cue = rig.transitionCue(t, cue, constants.CuePendingReview)
		cue = rig.transitionCue(t, cue, constants.CueApproved)

		start := make(chan struct{})
		results := make([]error, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			_, err := rig.updateStatus(device, model.DeviceStatusInspectionHold)
			results[0] = err
		}()
		go func() {
			defer wg.Done()
			<-start
			_, err := rig.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: cue.Version, Reason: "race lock"}, constants.CueLocked, freezeActor)
			results[1] = err
		}()
		close(start)
		wg.Wait()

		if (results[0] == nil) == (results[1] == nil) {
			t.Fatalf("attempt %d: exactly one change must succeed, got device=%v cue=%v", attempt, results[0], results[1])
		}
		storedDevice, err := rig.devices.Get(device.ID)
		if err != nil {
			t.Fatalf("reload device: %v", err)
		}
		storedCue, err := rig.cues.Get(cue.ID)
		if err != nil {
			t.Fatalf("reload cue: %v", err)
		}
		if results[0] == nil {
			if storedDevice.DeviceStatus != model.DeviceStatusInspectionHold || storedCue.CueStatus != constants.CueApproved {
				t.Fatalf("attempt %d: device won but state is inconsistent: device=%s cue=%s", attempt, storedDevice.DeviceStatus, storedCue.CueStatus)
			}
		} else if storedCue.CueStatus != constants.CueLocked || storedDevice.DeviceStatus != model.DeviceStatusAvailable {
			t.Fatalf("attempt %d: cue won but state is inconsistent: device=%s cue=%s", attempt, storedDevice.DeviceStatus, storedCue.CueStatus)
		}
	}
}
