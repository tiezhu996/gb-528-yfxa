package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/model"
)

type CueAction struct {
	DeviceID      uint    `json:"device_id" binding:"required,gte=1"`
	StartOffsetMS int64   `json:"start_offset_ms" binding:"gte=0,lte=86400000"`
	DurationMS    int64   `json:"duration_ms" binding:"required,gt=0,lte=86400000"`
	FromPositionM float64 `json:"from_position_m" binding:"gte=-20,lte=200"`
	ToPositionM   float64 `json:"to_position_m" binding:"gte=-20,lte=200"`
	LoadKG        float64 `json:"load_kg" binding:"gte=0,lte=100000"`
}

type CreateCueRequest struct {
	CueCode       string      `json:"cue_code" binding:"required,min=2,max=48"`
	Name          string      `json:"name" binding:"required,min=3,max=120"`
	SequenceNo    int         `json:"sequence_no" binding:"required,gte=1,lte=100000"`
	StartOffsetMS int64       `json:"start_offset_ms" binding:"gte=0,lte=86400000"`
	DurationMS    int64       `json:"duration_ms" binding:"required,gt=0,lte=86400000"`
	Actions       []CueAction `json:"actions" binding:"required,min=1,max=24,dive"`
	DependencyIDs []uint      `json:"dependency_ids" binding:"max=40,dive,gte=1"`
}

type UpdateCueRequest struct {
	Name          string      `json:"name" binding:"required,min=3,max=120"`
	SequenceNo    int         `json:"sequence_no" binding:"required,gte=1,lte=100000"`
	StartOffsetMS int64       `json:"start_offset_ms" binding:"gte=0,lte=86400000"`
	DurationMS    int64       `json:"duration_ms" binding:"required,gt=0,lte=86400000"`
	Actions       []CueAction `json:"actions" binding:"required,min=1,max=24,dive"`
	DependencyIDs []uint      `json:"dependency_ids" binding:"max=40,dive,gte=1"`
	Version       uint        `json:"version" binding:"required,gte=1"`
}

type CueTransitionRequest struct {
	Version uint   `json:"version" binding:"required,gte=1"`
	Reason  string `json:"reason" binding:"required,min=4,max=500"`
}

type CueDefinitionResponse struct {
	ID            uint                `json:"id"`
	CueCode       string              `json:"cue_code"`
	Name          string              `json:"name"`
	SequenceNo    int                 `json:"sequence_no"`
	StartOffsetMS int64               `json:"start_offset_ms"`
	DurationMS    int64               `json:"duration_ms"`
	CueStatus     constants.CueStatus `json:"cue_status"`
	Version       uint                `json:"version"`
	CreatedBy     uint                `json:"created_by"`
	ApprovedBy    *uint               `json:"approved_by"`
	Actions       []CueAction         `json:"actions"`
	DependencyIDs []uint              `json:"dependency_ids"`
	ReviewNote    string              `json:"review_note"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

func CueFromModel(item model.CueDefinition) (CueDefinitionResponse, error) {
	actions := []CueAction{}
	if err := json.Unmarshal(item.ActionsJSON, &actions); err != nil {
		return CueDefinitionResponse{}, fmt.Errorf("decode cue %s actions: %w", item.CueCode, err)
	}
	dependencies := []uint{}
	if err := json.Unmarshal(item.DependenciesJSON, &dependencies); err != nil {
		return CueDefinitionResponse{}, fmt.Errorf("decode cue %s dependencies: %w", item.CueCode, err)
	}
	return CueDefinitionResponse{ID: item.ID, CueCode: item.CueCode, Name: item.Name, SequenceNo: item.SequenceNo, StartOffsetMS: item.StartOffsetMS, DurationMS: item.DurationMS, CueStatus: constants.CueStatus(item.CueStatus), Version: item.Version, CreatedBy: item.CreatedBy, ApprovedBy: item.ApprovedBy, Actions: actions, DependencyIDs: dependencies, ReviewNote: item.ReviewNote, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, nil
}
