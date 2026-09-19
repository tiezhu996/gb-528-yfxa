package model

import (
	"time"

	"gorm.io/datatypes"
)

type InterlockRule struct {
	ID            uint           `gorm:"primaryKey"`
	RuleCode      string         `gorm:"size:48;uniqueIndex;not null"`
	RuleType      string         `gorm:"size:32;not null;check:chk_rule_type,rule_type IN ('load_limit','speed_limit','travel_limit','zone_exclusion','dependency_guard')"`
	DeviceIDsJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	ThresholdJSON datatypes.JSON `gorm:"type:jsonb;not null"`
	Severity      string         `gorm:"size:16;not null;check:chk_rule_severity,severity IN ('warning','blocker')"`
	Enabled       bool           `gorm:"not null;default:true"`
	RuleVersion   uint           `gorm:"not null;default:1"`
	Explanation   string         `gorm:"size:600;not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (InterlockRule) TableName() string { return "interlock_rules" }
