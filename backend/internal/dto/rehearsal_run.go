package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/interlock"
	"stage-rigging-cue-interlock/backend/internal/model"
)

type RunRehearsalRequest struct {
	CueIDs []uint `json:"cue_ids" binding:"required,min=1,max=40,dive,gte=1"`
}

type RunTransitionRequest struct {
	Version uint   `json:"version" binding:"required,gte=1"`
	Reason  string `json:"reason" binding:"required,min=4,max=500"`
}

type ReviewRunRequest struct {
	Version  uint   `json:"version" binding:"required,gte=1"`
	Decision string `json:"decision" binding:"required,oneof=approve reject"`
	Reason   string `json:"reason" binding:"required,min=4,max=500"`
}

type TimelineSnapshot struct {
	CueSetVersion  string                    `json:"cue_set_version"`
	CueIDs         []uint                    `json:"cue_ids"`
	Cues           []interlock.GraphCue      `json:"cues"`
	CueInputs      []interlock.CueInput      `json:"cue_inputs"`
	DeviceInputs   []interlock.DeviceInput   `json:"device_inputs"`
	RuleInputs     []interlock.RuleInput     `json:"rule_inputs"`
	Timeline       []interlock.TimelineEvent `json:"timeline"`
	RuleVersions   map[string]uint           `json:"rule_versions"`
	TimelineStepMS int64                     `json:"timeline_step_ms"`
	Assumptions    []string                  `json:"assumptions"`
}

type RehearsalRunResponse struct {
	ID               uint                        `json:"id"`
	CueSetVersion    string                      `json:"cue_set_version"`
	RunStatus        constants.RehearsalStatus   `json:"run_status"`
	TimelineSnapshot TimelineSnapshot            `json:"timeline_snapshot"`
	RuleResults      []interlock.RuleEvidence    `json:"rule_results"`
	CollisionWindows []interlock.CollisionWindow `json:"collision_windows"`
	HighestSeverity  constants.InterlockResult   `json:"highest_severity"`
	StartedBy        uint                        `json:"started_by"`
	ReviewedBy       *uint                       `json:"reviewed_by"`
	ReviewReason     string                      `json:"review_reason"`
	Version          uint                        `json:"version"`
	FinishedAt       time.Time                   `json:"finished_at"`
	ReviewedAt       *time.Time                  `json:"reviewed_at"`
	CreatedAt        time.Time                   `json:"created_at"`
}

func RunFromModel(item model.RehearsalRun) (RehearsalRunResponse, error) {
	snapshot := TimelineSnapshot{}
	if err := json.Unmarshal(item.TimelineSnapshotJSON, &snapshot); err != nil {
		return RehearsalRunResponse{}, fmt.Errorf("decode run %d timeline snapshot: %w", item.ID, err)
	}
	results := []interlock.RuleEvidence{}
	if err := json.Unmarshal(item.RuleResultsJSON, &results); err != nil {
		return RehearsalRunResponse{}, fmt.Errorf("decode run %d rule results: %w", item.ID, err)
	}
	windows := []interlock.CollisionWindow{}
	if err := json.Unmarshal(item.CollisionWindowsJSON, &windows); err != nil {
		return RehearsalRunResponse{}, fmt.Errorf("decode run %d collision windows: %w", item.ID, err)
	}
	return RehearsalRunResponse{ID: item.ID, CueSetVersion: item.CueSetVersion, RunStatus: constants.RehearsalStatus(item.RunStatus), TimelineSnapshot: snapshot, RuleResults: results, CollisionWindows: windows, HighestSeverity: constants.InterlockResult(item.HighestSeverity), StartedBy: item.StartedBy, ReviewedBy: item.ReviewedBy, ReviewReason: item.ReviewReason, Version: item.Version, FinishedAt: item.FinishedAt, ReviewedAt: item.ReviewedAt, CreatedAt: item.CreatedAt}, nil
}
