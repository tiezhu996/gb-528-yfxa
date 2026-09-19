package service

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
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

var freezeDBCounter int64

type freezeFixture struct {
	db       *gorm.DB
	devices  *RiggingDeviceService
	cues     *CueDefinitionService
	actor    audit.ActorContext
	reviewer audit.ActorContext
}

func newFreezeFixture(t *testing.T) *freezeFixture {
	t.Helper()
	index := atomic.AddInt64(&freezeDBCounter, 1)
	dsn := fmt.Sprintf("file:freeze-test-%d?mode=memory&cache=shared", index)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sqlite handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.RiggingDevice{}, &model.CueDefinition{}, &model.InterlockRule{}, &model.RehearsalRun{}, &audit.Event{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	auditRepository := audit.NewRepository(db)
	deviceRepository := repository.NewRiggingDeviceRepository(db, auditRepository)
	cueRepository := repository.NewCueDefinitionRepository(db, auditRepository)
	ruleRepository := repository.NewInterlockRuleRepository(db, auditRepository)
	return &freezeFixture{
		db:       db,
		devices:  NewRiggingDeviceService(db, deviceRepository, cueRepository, ruleRepository),
		cues:     NewCueDefinitionService(db, cueRepository, deviceRepository),
		actor:    audit.ActorContext{ID: 1, Username: "programmer", RequestID: "req-test"},
		reviewer: audit.ActorContext{ID: 2, Username: "reviewer", RequestID: "req-test"},
	}
}

func createTestDevice(t *testing.T, f *freezeFixture, code string) dto.RiggingDeviceResponse {
	t.Helper()
	device, err := f.devices.Create(dto.CreateRiggingDeviceRequest{DeviceCode: code, Name: "Test hoist " + code, DeviceType: "point_hoist", MaxLoadKG: 500, MaxSpeedMS: 0.5, TravelMinM: 2, TravelMaxM: 12, SafetyZone: "zone-x", DeviceStatus: "available"}, f.actor)
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	return device
}

func createTestCue(t *testing.T, f *freezeFixture, cueCode string, deviceID uint) dto.CueDefinitionResponse {
	t.Helper()
	cue, err := f.cues.Create(dto.CreateCueRequest{CueCode: cueCode, Name: "Test cue " + cueCode, SequenceNo: 10, StartOffsetMS: 0, DurationMS: 8000, Actions: []dto.CueAction{{DeviceID: deviceID, StartOffsetMS: 0, DurationMS: 8000, FromPositionM: 10, ToPositionM: 6, LoadKG: 200}}, DependencyIDs: []uint{}}, f.actor)
	if err != nil {
		t.Fatalf("create cue: %v", err)
	}
	return cue
}

func lockCue(t *testing.T, f *freezeFixture, cue dto.CueDefinitionResponse) dto.CueDefinitionResponse {
	t.Helper()
	reason := "offline freeze test review with enough detail"
	submitReq := dto.CueTransitionRequest{Version: cue.Version, Reason: reason}
	submitted, err := f.cues.Transition(cue.ID, submitReq, constants.CuePendingReview, f.actor)
	if err != nil {
		t.Fatalf("submit cue: %v", err)
	}
	approved, err := f.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: submitted.Version, Reason: reason}, constants.CueApproved, f.reviewer)
	if err != nil {
		t.Fatalf("approve cue: %v", err)
	}
	locked, err := f.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: approved.Version, Reason: reason}, constants.CueLocked, f.reviewer)
	if err != nil {
		t.Fatalf("lock cue: %v", err)
	}
	return locked
}

func freezeErrorDetails(t *testing.T, err error) map[string]any {
	t.Helper()
	var appErr *util.AppError
	if !asAppError(err, &appErr) {
		t.Fatalf("expected app error, got %T: %v", err, err)
	}
	if appErr.Code != "DEVICE_MAINTENANCE_BLOCKED" {
		t.Fatalf("expected DEVICE_MAINTENANCE_BLOCKED, got %s", appErr.Code)
	}
	details, ok := appErr.Details.(map[string]any)
	if !ok {
		t.Fatalf("expected details map, got %T", appErr.Details)
	}
	return details
}

func asAppError(err error, target **util.AppError) bool {
	for err != nil {
		if appErr, ok := err.(*util.AppError); ok {
			*target = appErr
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func TestMaintenanceFreezeRejectsWhenLockedCueReferencesDevice(t *testing.T) {
	f := newFreezeFixture(t)
	device := createTestDevice(t, f, "HOIST-FRZ-01")
	cue := createTestCue(t, f, "Q-FRZ-010", device.ID)
	lockCue(t, f, cue)

	_, err := f.devices.Update(device.ID, dto.UpdateRiggingDeviceRequest{Name: device.Name, DeviceType: "point_hoist", MaxLoadKG: device.MaxLoadKG, MaxSpeedMS: device.MaxSpeedMS, TravelMinM: device.TravelMinM, TravelMaxM: device.TravelMaxM, SafetyZone: device.SafetyZone, DeviceStatus: "inspection_hold", Version: device.Version}, f.actor)
	if err == nil {
		t.Fatal("maintenance transition must be rejected while a locked cue references the device")
	}
	details := freezeErrorDetails(t, err)
	codes, _ := details["locked_cue_codes"].([]string)
	if len(codes) != 1 || codes[0] != "Q-FRZ-010" {
		t.Fatalf("expected blocking cue code Q-FRZ-010, got %v", codes)
	}

	stored, getErr := f.devices.Get(device.ID)
	if getErr != nil {
		t.Fatalf("reload device: %v", getErr)
	}
	if stored.DeviceStatus != "available" || stored.Version != device.Version {
		t.Fatalf("device must remain unchanged after rejection, got status=%s version=%d", stored.DeviceStatus, stored.Version)
	}
	if !stored.MaintenanceFreeze.Blocked {
		t.Fatal("device detail must expose the blocking freeze reason")
	}
	if len(stored.MaintenanceFreeze.LockedCues) != 1 || stored.MaintenanceFreeze.LockedCues[0].CueCode != "Q-FRZ-010" {
		t.Fatalf("maintenance freeze must name the locked cue, got %+v", stored.MaintenanceFreeze.LockedCues)
	}
}

func TestMaintenanceAllowedAfterLockedCueArchived(t *testing.T) {
	f := newFreezeFixture(t)
	device := createTestDevice(t, f, "HOIST-FRZ-02")
	locked := lockCue(t, f, createTestCue(t, f, "Q-FRZ-020", device.ID))

	if _, err := f.cues.Transition(locked.ID, dto.CueTransitionRequest{Version: locked.Version, Reason: "archive locked reference before scheduled maintenance"}, constants.CueArchived, f.reviewer); err != nil {
		t.Fatalf("archive cue: %v", err)
	}
	updated, err := f.devices.Update(device.ID, dto.UpdateRiggingDeviceRequest{Name: device.Name, DeviceType: "point_hoist", MaxLoadKG: device.MaxLoadKG, MaxSpeedMS: device.MaxSpeedMS, TravelMinM: device.TravelMinM, TravelMaxM: device.TravelMaxM, SafetyZone: device.SafetyZone, DeviceStatus: "retired", Version: device.Version}, f.actor)
	if err != nil {
		t.Fatalf("retirement must succeed once the locked cue is archived: %v", err)
	}
	if updated.DeviceStatus != "retired" {
		t.Fatalf("expected retired, got %s", updated.DeviceStatus)
	}
}

func TestMaintenanceAllowedAfterLockedCueRevisedToNewDraft(t *testing.T) {
	f := newFreezeFixture(t)
	device := createTestDevice(t, f, "HOIST-FRZ-03")
	locked := lockCue(t, f, createTestCue(t, f, "Q-FRZ-030", device.ID))

	revised, err := f.cues.Transition(locked.ID, dto.CueTransitionRequest{Version: locked.Version, Reason: "revise locked version into a new draft for maintenance"}, constants.CueDraft, f.reviewer)
	if err != nil {
		t.Fatalf("revise cue: %v", err)
	}
	if revised.CueStatus != constants.CueDraft || revised.ApprovedBy != nil {
		t.Fatalf("revised cue must be a clean draft, got status=%s approved_by=%v", revised.CueStatus, revised.ApprovedBy)
	}
	updated, err := f.devices.Update(device.ID, dto.UpdateRiggingDeviceRequest{Name: device.Name, DeviceType: "point_hoist", MaxLoadKG: device.MaxLoadKG, MaxSpeedMS: device.MaxSpeedMS, TravelMinM: device.TravelMinM, TravelMaxM: device.TravelMaxM, SafetyZone: device.SafetyZone, DeviceStatus: "inspection_hold", Version: device.Version}, f.actor)
	if err != nil {
		t.Fatalf("maintenance must succeed once the cue is a new draft: %v", err)
	}
	if updated.DeviceStatus != "inspection_hold" {
		t.Fatalf("expected inspection_hold, got %s", updated.DeviceStatus)
	}
}

func TestCueCannotBeLockedWhenDeviceAlreadyFrozen(t *testing.T) {
	f := newFreezeFixture(t)
	device := createTestDevice(t, f, "HOIST-FRZ-04")
	cue := createTestCue(t, f, "Q-FRZ-040", device.ID)

	if _, err := f.devices.Update(device.ID, dto.UpdateRiggingDeviceRequest{Name: device.Name, DeviceType: "point_hoist", MaxLoadKG: device.MaxLoadKG, MaxSpeedMS: device.MaxSpeedMS, TravelMinM: device.TravelMinM, TravelMaxM: device.TravelMaxM, SafetyZone: device.SafetyZone, DeviceStatus: "inspection_hold", Version: device.Version}, f.actor); err != nil {
		t.Fatalf("freeze device without locked cues: %v", err)
	}
	reason := "attempt lock against a device already in maintenance"
	submitted, err := f.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: cue.Version, Reason: reason}, constants.CuePendingReview, f.actor)
	if err != nil {
		t.Fatalf("submit cue: %v", err)
	}
	approved, err := f.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: submitted.Version, Reason: reason}, constants.CueApproved, f.reviewer)
	if err != nil {
		t.Fatalf("approve cue: %v", err)
	}
	if _, err := f.cues.Transition(cue.ID, dto.CueTransitionRequest{Version: approved.Version, Reason: reason}, constants.CueLocked, f.reviewer); err == nil {
		t.Fatal("locking a cue that references a frozen device must fail")
	} else {
		var appErr *util.AppError
		if !asAppError(err, &appErr) || appErr.Code != "CUE_DEVICE_UNAVAILABLE" {
			t.Fatalf("expected CUE_DEVICE_UNAVAILABLE, got %v", err)
		}
	}
	stored, _ := f.cues.Get(cue.ID)
	if stored.CueStatus != constants.CueApproved {
		t.Fatalf("failed lock must not leave a half update, status=%s", stored.CueStatus)
	}
}

func TestLimitOnlyUpdateDoesNotTriggerFreeze(t *testing.T) {
	f := newFreezeFixture(t)
	device := createTestDevice(t, f, "HOIST-FRZ-05")
	lockCue(t, f, createTestCue(t, f, "Q-FRZ-050", device.ID))

	updated, err := f.devices.Update(device.ID, dto.UpdateRiggingDeviceRequest{Name: device.Name, DeviceType: "point_hoist", MaxLoadKG: 420, MaxSpeedMS: device.MaxSpeedMS, TravelMinM: device.TravelMinM, TravelMaxM: device.TravelMaxM, SafetyZone: device.SafetyZone, DeviceStatus: "available", Version: device.Version}, f.actor)
	if err != nil {
		t.Fatalf("limit revision while available must not trigger freeze: %v", err)
	}
	if updated.MaxLoadKG != 420 {
		t.Fatalf("expected updated load 420, got %v", updated.MaxLoadKG)
	}
}

func TestFreezeReviewListsEnabledInterlockRules(t *testing.T) {
	f := newFreezeFixture(t)
	device := createTestDevice(t, f, "HOIST-FRZ-06")
	ruleIDs := mustMarshalIDs(t, []uint{device.ID})
	if err := f.db.Create(&model.InterlockRule{RuleCode: "FRZ-RULE-01", RuleType: "load_limit", DeviceIDsJSON: datatypes.JSON(ruleIDs), ThresholdJSON: datatypes.JSON([]byte(`{"max_load_kg":300}`)), Severity: "blocker", Enabled: true, RuleVersion: 1, Explanation: "freeze review scope rule"}).Error; err != nil {
		t.Fatalf("create rule: %v", err)
	}

	detail, err := f.devices.Get(device.ID)
	if err != nil {
		t.Fatalf("get device: %v", err)
	}
	if detail.MaintenanceFreeze.Blocked {
		t.Fatal("enabled rules alone must not hard-block maintenance")
	}
	if len(detail.MaintenanceFreeze.EnabledInterlockRules) != 1 || detail.MaintenanceFreeze.EnabledInterlockRules[0].RuleCode != "FRZ-RULE-01" {
		t.Fatalf("freeze review must list the enabled rule, got %+v", detail.MaintenanceFreeze.EnabledInterlockRules)
	}
}

func mustMarshalIDs(t *testing.T, ids []uint) []byte {
	t.Helper()
	encoded, err := json.Marshal(ids)
	if err != nil {
		t.Fatalf("marshal ids: %v", err)
	}
	return encoded
}
