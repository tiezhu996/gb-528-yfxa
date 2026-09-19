package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/gorm"
)

type RehearsalRunRepository struct {
	db    *gorm.DB
	audit *audit.Repository
}

func NewRehearsalRunRepository(db *gorm.DB, auditRepository *audit.Repository) *RehearsalRunRepository {
	return &RehearsalRunRepository{db: db, audit: auditRepository}
}

func (r *RehearsalRunRepository) List(page, pageSize int, status, severity, search string) ([]model.RehearsalRun, int64, error) {
	query := r.db.Model(&model.RehearsalRun{})
	if status != "" {
		query = query.Where("run_status = ?", status)
	}
	if severity != "" {
		query = query.Where("highest_severity = ?", severity)
	}
	if search != "" {
		query = query.Where("LOWER(cue_set_version) LIKE ?", "%"+strings.ToLower(search)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count rehearsal runs: %w", err)
	}
	var items []model.RehearsalRun
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list rehearsal runs: %w", err)
	}
	return items, total, nil
}

func (r *RehearsalRunRepository) Get(id uint) (model.RehearsalRun, error) {
	var item model.RehearsalRun
	if err := r.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.RehearsalRun{}, util.NotFound("RUN_NOT_FOUND", "rehearsal run was not found")
		}
		return model.RehearsalRun{}, fmt.Errorf("get rehearsal run: %w", err)
	}
	return item, nil
}

func (r *RehearsalRunRepository) Create(item *model.RehearsalRun, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return fmt.Errorf("create immutable rehearsal run: %w", err)
		}
		event.EntityID = item.ID
		return r.audit.WithTx(tx).Record(event)
	})
}

func (r *RehearsalRunRepository) Transition(id, expectedVersion uint, from, to constants.RehearsalStatus, reviewerID *uint, reason string, event audit.Event) (model.RehearsalRun, error) {
	var updated model.RehearsalRun
	err := r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"run_status": to, "version": expectedVersion + 1, "review_reason": reason}
		if reviewerID != nil {
			now := time.Now().UTC()
			updates["reviewed_by"] = reviewerID
			updates["reviewed_at"] = now
		}
		result := tx.Model(&model.RehearsalRun{}).Where("id = ? AND version = ? AND run_status = ?", id, expectedVersion, from).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("transition rehearsal run: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return util.Conflict("RUN_VERSION_CONFLICT", "rehearsal run state or version changed concurrently", nil)
		}
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload rehearsal run: %w", err)
		}
		return nil
	})
	return updated, err
}
