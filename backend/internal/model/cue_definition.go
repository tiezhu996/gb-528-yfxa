package model

import (
	"time"

	"gorm.io/datatypes"
)

type CueDefinition struct {
	ID               uint   `gorm:"primaryKey"`
	CueCode          string `gorm:"size:48;uniqueIndex;not null"`
	Name             string `gorm:"size:120;not null"`
	SequenceNo       int    `gorm:"not null;index"`
	StartOffsetMS    int64  `gorm:"not null;check:chk_cue_start,start_offset_ms >= 0"`
	DurationMS       int64  `gorm:"not null;check:chk_cue_duration,duration_ms > 0"`
	CueStatus        string `gorm:"size:24;not null;check:chk_cue_status,cue_status IN ('draft','pending_review','approved','locked','archived')"`
	Version          uint   `gorm:"not null;default:1"`
	CreatedBy        uint   `gorm:"not null"`
	ApprovedBy       *uint
	ActionsJSON      datatypes.JSON `gorm:"type:jsonb;not null"`
	DependenciesJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	ReviewNote       string         `gorm:"size:500"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (CueDefinition) TableName() string { return "cue_definitions" }
