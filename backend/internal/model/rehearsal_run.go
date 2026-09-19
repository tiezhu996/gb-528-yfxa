package model

import (
	"time"

	"gorm.io/datatypes"
)

type RehearsalRun struct {
	ID                   uint           `gorm:"primaryKey"`
	CueSetVersion        string         `gorm:"size:160;not null;index"`
	RunStatus            string         `gorm:"size:32;not null;check:chk_run_status,run_status IN ('evaluated','blocked','pending_review','approved_for_rehearsal','rejected')"`
	TimelineSnapshotJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	RuleResultsJSON      datatypes.JSON `gorm:"type:jsonb;not null"`
	CollisionWindowsJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	HighestSeverity      string         `gorm:"size:16;not null;check:chk_run_severity,highest_severity IN ('pass','warning','blocker','invalid')"`
	StartedBy            uint           `gorm:"not null"`
	ReviewedBy           *uint
	ReviewReason         string    `gorm:"size:500"`
	Version              uint      `gorm:"not null;default:1"`
	FinishedAt           time.Time `gorm:"not null"`
	ReviewedAt           *time.Time
	CreatedAt            time.Time
}

func (RehearsalRun) TableName() string { return "rehearsal_runs" }
