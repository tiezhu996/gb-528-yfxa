package repository

import (
	"errors"
	"fmt"
	"strings"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/gorm"
)

type InterlockRuleRepository struct {
	db    *gorm.DB
	audit *audit.Repository
}

func NewInterlockRuleRepository(db *gorm.DB, auditRepository *audit.Repository) *InterlockRuleRepository {
	return &InterlockRuleRepository{db: db, audit: auditRepository}
}

func (r *InterlockRuleRepository) List(page, pageSize int, ruleType, severity, search string) ([]model.InterlockRule, int64, error) {
	query := r.db.Model(&model.InterlockRule{})
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}
	if severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if search != "" {
		term := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(rule_code) LIKE ? OR LOWER(explanation) LIKE ?", term, term)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count interlock rules: %w", err)
	}
	var items []model.InterlockRule
	if err := query.Order("rule_code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list interlock rules: %w", err)
	}
	return items, total, nil
}

func (r *InterlockRuleRepository) Enabled() ([]model.InterlockRule, error) {
	var items []model.InterlockRule
	if err := r.db.Where("enabled = ?", true).Order("rule_code ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list enabled interlock rules: %w", err)
	}
	return items, nil
}

func (r *InterlockRuleRepository) Get(id uint) (model.InterlockRule, error) {
	var item model.InterlockRule
	if err := r.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.InterlockRule{}, util.NotFound("RULE_NOT_FOUND", "interlock rule was not found")
		}
		return model.InterlockRule{}, fmt.Errorf("get interlock rule: %w", err)
	}
	return item, nil
}

func (r *InterlockRuleRepository) Create(item *model.InterlockRule, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return util.Conflict("RULE_CODE_CONFLICT", "rule_code already exists", err)
			}
			return fmt.Errorf("create interlock rule: %w", err)
		}
		event.EntityID = item.ID
		return r.audit.WithTx(tx).Record(event)
	})
}

func (r *InterlockRuleRepository) Update(item *model.InterlockRule, expectedVersion uint, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"device_ids_json": item.DeviceIDsJSON, "threshold_json": item.ThresholdJSON, "severity": item.Severity, "enabled": item.Enabled, "explanation": item.Explanation, "rule_version": expectedVersion + 1}
		result := tx.Model(&model.InterlockRule{}).Where("id = ? AND rule_version = ?", item.ID, expectedVersion).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update interlock rule: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return util.Conflict("RULE_VERSION_CONFLICT", "interlock rule changed since it was loaded", nil)
		}
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		item.RuleVersion = expectedVersion + 1
		return nil
	})
}

func (r *InterlockRuleRepository) Toggle(id, expectedVersion uint, enabled bool, event audit.Event) (model.InterlockRule, error) {
	var updated model.InterlockRule
	err := r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.InterlockRule{}).Where("id = ? AND rule_version = ?", id, expectedVersion).Updates(map[string]any{"enabled": enabled, "rule_version": expectedVersion + 1})
		if result.Error != nil {
			return fmt.Errorf("toggle interlock rule: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return util.Conflict("RULE_VERSION_CONFLICT", "interlock rule changed since it was loaded", nil)
		}
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload toggled rule: %w", err)
		}
		return nil
	})
	return updated, err
}
