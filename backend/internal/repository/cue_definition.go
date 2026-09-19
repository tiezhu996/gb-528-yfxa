package repository

import (
	"errors"
	"fmt"
	"strings"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/model"
	"stage-rigging-cue-interlock/backend/internal/util"

	"gorm.io/gorm"
)

type CueDefinitionRepository struct {
	db    *gorm.DB
	audit *audit.Repository
}

func NewCueDefinitionRepository(db *gorm.DB, auditRepository *audit.Repository) *CueDefinitionRepository {
	return &CueDefinitionRepository{db: db, audit: auditRepository}
}

func (r *CueDefinitionRepository) List(page, pageSize int, status, search string) ([]model.CueDefinition, int64, error) {
	query := r.db.Model(&model.CueDefinition{})
	if status != "" {
		query = query.Where("cue_status = ?", status)
	}
	if search != "" {
		term := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(cue_code) LIKE ? OR LOWER(name) LIKE ?", term, term)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count cue definitions: %w", err)
	}
	var items []model.CueDefinition
	if err := query.Order("sequence_no ASC, cue_code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list cue definitions: %w", err)
	}
	return items, total, nil
}

func (r *CueDefinitionRepository) Get(id uint) (model.CueDefinition, error) {
	var item model.CueDefinition
	if err := r.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.CueDefinition{}, util.NotFound("CUE_NOT_FOUND", "cue definition was not found")
		}
		return model.CueDefinition{}, fmt.Errorf("get cue definition: %w", err)
	}
	return item, nil
}

func (r *CueDefinitionRepository) ByIDs(ids []uint) ([]model.CueDefinition, error) {
	var items []model.CueDefinition
	if err := r.db.Where("id IN ?", ids).Order("sequence_no ASC, cue_code ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("get cues by ids: %w", err)
	}
	if len(items) != len(uniqueIDs(ids)) {
		return nil, util.Unprocessable("CUE_SET_INVALID", "one or more selected cues do not exist", map[string]any{"requested_ids": ids, "found_count": len(items)})
	}
	return items, nil
}

func (r *CueDefinitionRepository) Create(item *model.CueDefinition, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return util.Conflict("CUE_CODE_CONFLICT", "cue_code already exists", err)
			}
			return fmt.Errorf("create cue definition: %w", err)
		}
		event.EntityID = item.ID
		return r.audit.WithTx(tx).Record(event)
	})
}

func (r *CueDefinitionRepository) Update(item *model.CueDefinition, expectedVersion uint, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"name": item.Name, "sequence_no": item.SequenceNo, "start_offset_ms": item.StartOffsetMS, "duration_ms": item.DurationMS, "actions_json": item.ActionsJSON, "dependencies_json": item.DependenciesJSON, "version": expectedVersion + 1}
		result := tx.Model(&model.CueDefinition{}).Where("id = ? AND version = ? AND cue_status = ?", item.ID, expectedVersion, constants.CueDraft).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update draft cue: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return util.Conflict("CUE_VERSION_CONFLICT", "cue changed or is no longer editable as a draft", nil)
		}
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		item.Version = expectedVersion + 1
		return nil
	})
}

func (r *CueDefinitionRepository) Transition(id uint, expectedVersion uint, from, to constants.CueStatus, reviewerID *uint, note string, event audit.Event) (model.CueDefinition, error) {
	var updated model.CueDefinition
	err := r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"cue_status": to, "version": expectedVersion + 1, "review_note": note}
		if to == constants.CueApproved {
			updates["approved_by"] = reviewerID
		}
		if to == constants.CueDraft {
			updates["approved_by"] = nil
		}
		result := tx.Model(&model.CueDefinition{}).Where("id = ? AND version = ? AND cue_status = ?", id, expectedVersion, from).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("transition cue: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return util.Conflict("CUE_VERSION_CONFLICT", "cue state or version changed concurrently", nil)
		}
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload transitioned cue: %w", err)
		}
		return nil
	})
	return updated, err
}
