package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/dto"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/repository"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CueDefinitionService struct {
	db      *gorm.DB
	cues    *repository.CueDefinitionRepository
	devices *repository.RiggingDeviceRepository
}

func NewCueDefinitionService(db *gorm.DB, cues *repository.CueDefinitionRepository, devices *repository.RiggingDeviceRepository) *CueDefinitionService {
	return &CueDefinitionService{db: db, cues: cues, devices: devices}
}

func (s *CueDefinitionService) List(page, pageSize int, status, search string) ([]dto.CueDefinitionResponse, int64, error) {
	items, total, err := s.cues.List(page, pageSize, status, search)
	if err != nil {
		return nil, 0, err
	}
	responses := make([]dto.CueDefinitionResponse, 0, len(items))
	for _, item := range items {
		response, mapErr := dto.CueFromModel(item)
		if mapErr != nil {
			return nil, 0, mapErr
		}
		responses = append(responses, response)
	}
	return responses, total, nil
}

func (s *CueDefinitionService) Get(id uint) (dto.CueDefinitionResponse, error) {
	item, err := s.cues.Get(id)
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	return dto.CueFromModel(item)
}

func (s *CueDefinitionService) Create(request dto.CreateCueRequest, actor audit.ActorContext) (dto.CueDefinitionResponse, error) {
	if err := s.validateCueInput(0, request.DurationMS, request.Actions, request.DependencyIDs); err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	actionsJSON, dependenciesJSON, err := marshalCueInput(request.Actions, request.DependencyIDs)
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	item := model.CueDefinition{CueCode: normalizeCode(request.CueCode), Name: strings.TrimSpace(request.Name), SequenceNo: request.SequenceNo, StartOffsetMS: request.StartOffsetMS, DurationMS: request.DurationMS, CueStatus: string(constants.CueDraft), Version: 1, CreatedBy: actor.ID, ActionsJSON: actionsJSON, DependenciesJSON: dependenciesJSON}
	after := cueSummary(item, request.Actions, request.DependencyIDs)
	if err := s.cues.Create(&item, audit.NewEvent(actor, "cue_definition.create", "cue_definition", 0, "{}", after)); err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	return dto.CueFromModel(item)
}

func (s *CueDefinitionService) Update(id uint, request dto.UpdateCueRequest, actor audit.ActorContext) (dto.CueDefinitionResponse, error) {
	if err := s.validateCueInput(id, request.DurationMS, request.Actions, request.DependencyIDs); err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	current, err := s.cues.Get(id)
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	if constants.CueStatus(current.CueStatus) != constants.CueDraft {
		return dto.CueDefinitionResponse{}, util.Unprocessable("CUE_NOT_EDITABLE", "only draft cues can be edited", map[string]any{"current_status": current.CueStatus})
	}
	beforeResponse, err := dto.CueFromModel(current)
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	actionsJSON, dependenciesJSON, err := marshalCueInput(request.Actions, request.DependencyIDs)
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	current.Name = strings.TrimSpace(request.Name)
	current.SequenceNo = request.SequenceNo
	current.StartOffsetMS = request.StartOffsetMS
	current.DurationMS = request.DurationMS
	current.ActionsJSON = actionsJSON
	current.DependenciesJSON = dependenciesJSON
	after := cueSummary(current, request.Actions, request.DependencyIDs)
	if err := s.cues.Update(&current, request.Version, audit.NewEvent(actor, "cue_definition.update", "cue_definition", id, util.SummaryJSON(beforeResponse), after)); err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	return dto.CueFromModel(current)
}

func (s *CueDefinitionService) Transition(id uint, request dto.CueTransitionRequest, target constants.CueStatus, actor audit.ActorContext) (dto.CueDefinitionResponse, error) {
	current, err := s.cues.Get(id)
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	from := constants.CueStatus(current.CueStatus)
	if !constants.CanTransitionCue(from, target) {
		return dto.CueDefinitionResponse{}, util.Unprocessable("INVALID_CUE_TRANSITION", fmt.Sprintf("cannot transition cue from %s to %s", from, target), map[string]any{"from": from, "to": target})
	}
	var reviewerID *uint
	if target == constants.CueApproved {
		reviewerID = &actor.ID
	}
	before := util.SummaryJSON(map[string]any{"cue_status": from, "version": current.Version, "approved_by": current.ApprovedBy})
	note := strings.TrimSpace(request.Reason)
	after := util.SummaryJSON(map[string]any{"cue_status": target, "version": request.Version + 1, "review_reason": note, "reviewer_id": reviewerID})
	event := audit.NewEvent(actor, "cue_definition.transition", "cue_definition", id, before, after)

	var updated model.CueDefinition
	if target == constants.CueLocked {
		updated, err = s.lockWithDeviceGuard(id, request.Version, from, reviewerID, note, event, current.ActionsJSON)
	} else {
		updated, err = s.runTransition(id, request.Version, from, target, reviewerID, note, event)
	}
	if err != nil {
		return dto.CueDefinitionResponse{}, err
	}
	return dto.CueFromModel(updated)
}

// runTransition executes a non-locking state migration in one transaction.
func (s *CueDefinitionService) runTransition(id uint, expectedVersion uint, from, target constants.CueStatus, reviewerID *uint, note string, event audit.Event) (model.CueDefinition, error) {
	var updated model.CueDefinition
	err := repository.RunInTransaction(s.db, func(tx *gorm.DB) error {
		var inner error
		updated, inner = s.cues.WithTx(tx).TransitionInTransaction(tx, id, expectedVersion, from, target, reviewerID, note, event)
		return inner
	})
	return updated, err
}

// lockWithDeviceGuard closes the device maintenance race: every device used by
// the cue is locked in id order and re-checked as available inside the same
// transaction that performs the cue lock. A concurrent freeze either wins
// (this guard sees inspection_hold/retired and rejects) or waits on the device
// row and then rejects via its own locked-cue scan, so no half update remains.
func (s *CueDefinitionService) lockWithDeviceGuard(id uint, expectedVersion uint, from constants.CueStatus, reviewerID *uint, note string, event audit.Event, actionsJSON datatypes.JSON) (model.CueDefinition, error) {
	var updated model.CueDefinition
	err := repository.RunInTransaction(s.db, func(tx *gorm.DB) error {
		actions := []dto.CueAction{}
		if err := json.Unmarshal(actionsJSON, &actions); err != nil {
			return fmt.Errorf("decode cue actions for lock guard: %w", err)
		}
		deviceIDs := make([]uint, 0, len(actions))
		for _, action := range actions {
			deviceIDs = append(deviceIDs, action.DeviceID)
		}
		lockedDevices, err := s.devices.WithTx(tx).LockByIDs(tx, deviceIDs)
		if err != nil {
			return err
		}
		unavailable := make([]string, 0)
		for _, device := range lockedDevices {
			if device.DeviceStatus != "available" {
				unavailable = append(unavailable, fmt.Sprintf("%s(%s)", device.DeviceCode, device.DeviceStatus))
			}
		}
		sort.Strings(unavailable)
		if len(unavailable) > 0 {
			return util.Unprocessable("CUE_DEVICE_UNAVAILABLE", "the cue can only be locked while every referenced device is available", map[string]any{"unavailable_devices": unavailable})
		}
		var inner error
		updated, inner = s.cues.WithTx(tx).TransitionInTransaction(tx, id, expectedVersion, from, constants.CueLocked, reviewerID, note, event)
		return inner
	})
	return updated, err
}

func (s *CueDefinitionService) validateCueInput(selfID uint, duration int64, actions []dto.CueAction, dependencyIDs []uint) error {
	seenDevices := map[uint]bool{}
	deviceIDs := make([]uint, 0, len(actions))
	for index, action := range actions {
		if action.StartOffsetMS+action.DurationMS > duration {
			return util.Unprocessable("ACTION_OUT_OF_CUE_BOUNDS", "action interval exceeds cue duration", map[string]any{"action_index": index, "cue_duration_ms": duration})
		}
		if seenDevices[action.DeviceID] {
			return util.Unprocessable("DUPLICATE_DEVICE_ACTION", "a cue can contain only one action per device", map[string]any{"device_id": action.DeviceID})
		}
		seenDevices[action.DeviceID] = true
		deviceIDs = append(deviceIDs, action.DeviceID)
	}
	devices, err := s.devices.ByIDs(deviceIDs)
	if err != nil {
		return err
	}
	for _, device := range devices {
		if device.DeviceStatus == "retired" {
			return util.Unprocessable("DEVICE_UNAVAILABLE", "retired devices cannot be added to a cue", map[string]any{"device_code": device.DeviceCode})
		}
	}
	seenDependencies := map[uint]bool{}
	for _, dependencyID := range dependencyIDs {
		if dependencyID == selfID && selfID != 0 {
			return util.Unprocessable("CUE_DEPENDENCY_CYCLE", "a cue cannot depend on itself", map[string]any{"evidence_path": []uint{selfID, selfID}})
		}
		if seenDependencies[dependencyID] {
			return util.Unprocessable("DUPLICATE_DEPENDENCY", "dependency_ids must be unique", map[string]any{"dependency_id": dependencyID})
		}
		seenDependencies[dependencyID] = true
	}
	if len(dependencyIDs) > 0 {
		if _, err := s.cues.ByIDs(dependencyIDs); err != nil {
			return err
		}
	}
	return nil
}

func marshalCueInput(actions []dto.CueAction, dependencies []uint) (datatypes.JSON, datatypes.JSON, error) {
	actionsJSON, err := json.Marshal(actions)
	if err != nil {
		return nil, nil, fmt.Errorf("encode cue actions: %w", err)
	}
	dependenciesJSON, err := json.Marshal(dependencies)
	if err != nil {
		return nil, nil, fmt.Errorf("encode cue dependencies: %w", err)
	}
	return datatypes.JSON(actionsJSON), datatypes.JSON(dependenciesJSON), nil
}

func cueSummary(item model.CueDefinition, actions []dto.CueAction, dependencies []uint) string {
	return util.SummaryJSON(map[string]any{"cue_code": item.CueCode, "sequence_no": item.SequenceNo, "start_offset_ms": item.StartOffsetMS, "duration_ms": item.DurationMS, "cue_status": item.CueStatus, "version": item.Version, "actions": actions, "dependency_ids": dependencies})
}
