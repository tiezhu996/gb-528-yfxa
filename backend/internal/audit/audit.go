package audit

import (
	"fmt"
	"strings"
	"time"

	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Event struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	RequestID     string    `gorm:"size:64;not null;index" json:"request_id"`
	ActorID       uint      `gorm:"not null;index" json:"actor_id"`
	ActorUsername string    `gorm:"size:64;not null" json:"actor_username"`
	Action        string    `gorm:"size:96;not null;index" json:"action"`
	EntityType    string    `gorm:"size:64;not null;index" json:"entity_type"`
	EntityID      uint      `gorm:"not null;index" json:"entity_id"`
	BeforeSummary string    `gorm:"type:text;not null" json:"before_summary"`
	AfterSummary  string    `gorm:"type:text;not null" json:"after_summary"`
	CreatedAt     time.Time `gorm:"not null;index" json:"created_at"`
}

func (Event) TableName() string { return "audit_events" }

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) WithTx(tx *gorm.DB) *Repository { return &Repository{db: tx} }

func (r *Repository) Record(event Event) error {
	event.BeforeSummary = truncate(event.BeforeSummary, 1600)
	event.AfterSummary = truncate(event.AfterSummary, 1600)
	if err := r.db.Create(&event).Error; err != nil {
		return fmt.Errorf("record audit event: %w", err)
	}
	return nil
}

func (r *Repository) List(page, pageSize int, entity, action, search string) ([]Event, int64, error) {
	query := r.db.Model(&Event{})
	if entity != "" {
		query = query.Where("entity_type = ?", entity)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if search != "" {
		term := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(actor_username) LIKE ? OR LOWER(request_id) LIKE ? OR LOWER(action) LIKE ?", term, term, term)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}
	var events []Event
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	return events, total, nil
}

type Handler struct{ repository *Repository }

func NewHandler(repository *Repository) *Handler { return &Handler{repository: repository} }

func (h *Handler) List(c *gin.Context) {
	page, pageSize := util.Pagination(c)
	events, total, err := h.repository.List(page, pageSize, c.Query("entity_type"), c.Query("action"), c.Query("search"))
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.Page(c, events, page, pageSize, total)
}

type ActorContext struct {
	ID        uint
	Username  string
	RequestID string
}

func ActorFromContext(c *gin.Context) ActorContext {
	actorID, actorName, _ := auth.Actor(c)
	return ActorContext{ID: actorID, Username: actorName, RequestID: util.RequestID(c)}
}

func NewEvent(actor ActorContext, action, entityType string, entityID uint, before, after string) Event {
	return Event{RequestID: actor.RequestID, ActorID: actor.ID, ActorUsername: actor.Username, Action: action, EntityType: entityType, EntityID: entityID, BeforeSummary: before, AfterSummary: after}
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
