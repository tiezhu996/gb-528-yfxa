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

type RiggingDeviceRepository struct {
	db    *gorm.DB
	audit *audit.Repository
}

func NewRiggingDeviceRepository(db *gorm.DB, auditRepository *audit.Repository) *RiggingDeviceRepository {
	return &RiggingDeviceRepository{db: db, audit: auditRepository}
}

func (r *RiggingDeviceRepository) List(page, pageSize int, status, search string) ([]model.RiggingDevice, int64, error) {
	query := r.db.Model(&model.RiggingDevice{})
	if status != "" {
		query = query.Where("device_status = ?", status)
	}
	if search != "" {
		term := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(device_code) LIKE ? OR LOWER(name) LIKE ? OR LOWER(safety_zone) LIKE ?", term, term, term)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count rigging devices: %w", err)
	}
	var items []model.RiggingDevice
	if err := query.Order("device_code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list rigging devices: %w", err)
	}
	return items, total, nil
}

func (r *RiggingDeviceRepository) All() ([]model.RiggingDevice, error) {
	var items []model.RiggingDevice
	if err := r.db.Order("device_code ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list all rigging devices: %w", err)
	}
	return items, nil
}

func (r *RiggingDeviceRepository) Get(id uint) (model.RiggingDevice, error) {
	var item model.RiggingDevice
	if err := r.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.RiggingDevice{}, util.NotFound("DEVICE_NOT_FOUND", "rigging device was not found")
		}
		return model.RiggingDevice{}, fmt.Errorf("get rigging device: %w", err)
	}
	return item, nil
}

func (r *RiggingDeviceRepository) ByIDs(ids []uint) ([]model.RiggingDevice, error) {
	if len(ids) == 0 {
		return []model.RiggingDevice{}, nil
	}
	var items []model.RiggingDevice
	if err := r.db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, fmt.Errorf("get rigging devices by ids: %w", err)
	}
	if len(items) != len(uniqueIDs(ids)) {
		return nil, util.Unprocessable("DEVICE_REFERENCE_INVALID", "one or more referenced devices do not exist", map[string]any{"requested_ids": ids, "found_count": len(items)})
	}
	return items, nil
}

func (r *RiggingDeviceRepository) Create(item *model.RiggingDevice, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return util.Conflict("DEVICE_CODE_CONFLICT", "device_code already exists", err)
			}
			return fmt.Errorf("create rigging device: %w", err)
		}
		event.EntityID = item.ID
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		return nil
	})
}

func (r *RiggingDeviceRepository) Update(item *model.RiggingDevice, expectedVersion uint, event audit.Event) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{"name": item.Name, "device_type": item.DeviceType, "max_load_kg": item.MaxLoadKG, "max_speed_ms": item.MaxSpeedMS, "travel_min_m": item.TravelMinM, "travel_max_m": item.TravelMaxM, "safety_zone": item.SafetyZone, "device_status": item.DeviceStatus, "version": expectedVersion + 1}
		result := tx.Model(&model.RiggingDevice{}).Where("id = ? AND version = ?", item.ID, expectedVersion).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update rigging device: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return util.Conflict("DEVICE_VERSION_CONFLICT", "device limits changed since they were loaded", nil)
		}
		if err := r.audit.WithTx(tx).Record(event); err != nil {
			return err
		}
		item.Version = expectedVersion + 1
		return nil
	})
}

func uniqueIDs(ids []uint) map[uint]struct{} {
	result := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result
}
