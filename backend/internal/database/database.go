package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"stage-rigging-cue-interlock/backend/internal/audit"
	"stage-rigging-cue-interlock/backend/internal/auth"
	"stage-rigging-cue-interlock/backend/internal/config"
	"stage-rigging-cue-interlock/backend/internal/constants"
	"stage-rigging-cue-interlock/backend/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBDriver == "postgres" {
		dialector = postgres.Open(cfg.DBDSN)
	} else {
		dialector = sqlite.Open(cfg.DBDSN)
	}
	db, err := gorm.Open(dialector, &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("open database handle: %w", err)
	}
	configurePool(sqlDB, cfg.DBDriver)
	if cfg.DBAutoMigrate {
		if err := migrate(db); err != nil {
			return nil, err
		}
		if err := seed(db); err != nil {
			return nil, err
		}
	}
	return db, nil
}

func configurePool(db *sql.DB, driver string) {
	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		return
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
}

func migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&auth.User{},
		&model.RiggingDevice{},
		&model.CueDefinition{},
		&model.InterlockRule{},
		&model.RehearsalRun{},
		&audit.Event{},
	); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}

func seed(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		users, err := seedUsers(tx)
		if err != nil {
			return err
		}
		devices, err := seedDevices(tx)
		if err != nil {
			return err
		}
		if err := seedCues(tx, users, devices); err != nil {
			return err
		}
		if err := seedRules(tx, devices); err != nil {
			return err
		}
		return nil
	})
}

func seedUsers(tx *gorm.DB) (map[string]auth.User, error) {
	definitions := []struct {
		username, password, displayName, role string
	}{
		{"programmer", "programmer123", "Mara Lin / Rigging Programmer", auth.RoleProgrammer},
		{"reviewer", "reviewer123", "Iris Cole / Safety Reviewer", auth.RoleSafetyReviewer},
		{"admin", "admin123", "System Steward", auth.RoleAdmin},
	}
	users := make(map[string]auth.User, len(definitions))
	for _, definition := range definitions {
		hash, err := bcrypt.GenerateFromPassword([]byte(definition.password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash seed password: %w", err)
		}
		user := auth.User{Username: definition.username, PasswordHash: string(hash), DisplayName: definition.displayName, Role: definition.role, Active: true}
		if err := tx.Where("username = ?", definition.username).FirstOrCreate(&user).Error; err != nil {
			return nil, fmt.Errorf("seed user %s: %w", definition.username, err)
		}
		users[definition.username] = user
	}
	return users, nil
}

func seedDevices(tx *gorm.DB) (map[string]model.RiggingDevice, error) {
	items := []model.RiggingDevice{
		{DeviceCode: "TRUSS-FOH-01", Name: "Front-of-house truss", DeviceType: "motorized_batten", MaxLoadKG: 800, MaxSpeedMS: 0.5, TravelMinM: 5, TravelMaxM: 16, SafetyZone: "overstage-a", DeviceStatus: "available", Version: 1},
		{DeviceCode: "LX-BRIDGE-02", Name: "Lighting bridge two", DeviceType: "motorized_bridge", MaxLoadKG: 600, MaxSpeedMS: 0.4, TravelMinM: 4, TravelMaxM: 14, SafetyZone: "overstage-b", DeviceStatus: "available", Version: 1},
		{DeviceCode: "SCENIC-CLOUD-03", Name: "Scenic cloud carrier", DeviceType: "scenic_carrier", MaxLoadKG: 450, MaxSpeedMS: 0.45, TravelMinM: 6, TravelMaxM: 18, SafetyZone: "overstage-a", DeviceStatus: "available", Version: 1},
	}
	result := make(map[string]model.RiggingDevice, len(items))
	for _, item := range items {
		if err := tx.Where("device_code = ?", item.DeviceCode).FirstOrCreate(&item).Error; err != nil {
			return nil, fmt.Errorf("seed device %s: %w", item.DeviceCode, err)
		}
		result[item.DeviceCode] = item
	}
	return result, nil
}

func seedCues(tx *gorm.DB, users map[string]auth.User, devices map[string]model.RiggingDevice) error {
	type action struct {
		DeviceID      uint    `json:"device_id"`
		StartOffsetMS int64   `json:"start_offset_ms"`
		DurationMS    int64   `json:"duration_ms"`
		FromPositionM float64 `json:"from_position_m"`
		ToPositionM   float64 `json:"to_position_m"`
		LoadKG        float64 `json:"load_kg"`
	}
	programmer := users["programmer"]
	reviewer := users["reviewer"]
	definitions := []struct {
		code, name string
		sequence   int
		start      int64
		duration   int64
		action     action
		dependsOn  string
	}{
		{"Q-010", "FOH truss rehearsal trim", 10, 0, 10000, action{devices["TRUSS-FOH-01"].ID, 0, 10000, 12, 8, 550}, ""},
		{"Q-020", "Lighting bridge rehearsal trim", 20, 11200, 10000, action{devices["LX-BRIDGE-02"].ID, 0, 10000, 10, 7, 400}, "Q-010"},
		{"Q-030", "Scenic cloud rehearsal trim", 30, 23400, 12000, action{devices["SCENIC-CLOUD-03"].ID, 0, 12000, 14, 9, 300}, "Q-020"},
	}
	created := make(map[string]model.CueDefinition, len(definitions))
	for _, definition := range definitions {
		actionsJSON, _ := json.Marshal([]action{definition.action})
		dependencies := []uint{}
		if dependency, ok := created[definition.dependsOn]; ok {
			dependencies = append(dependencies, dependency.ID)
		}
		dependenciesJSON, _ := json.Marshal(dependencies)
		cue := model.CueDefinition{CueCode: definition.code, Name: definition.name, SequenceNo: definition.sequence, StartOffsetMS: definition.start, DurationMS: definition.duration, CueStatus: string(constants.CueLocked), Version: 4, CreatedBy: programmer.ID, ApprovedBy: &reviewer.ID, ActionsJSON: datatypes.JSON(actionsJSON), DependenciesJSON: datatypes.JSON(dependenciesJSON), ReviewNote: "Seeded locked rehearsal reference; offline planning only."}
		if err := tx.Where("cue_code = ?", cue.CueCode).FirstOrCreate(&cue).Error; err != nil {
			return fmt.Errorf("seed cue %s: %w", cue.CueCode, err)
		}
		created[cue.CueCode] = cue
	}
	return nil
}

func seedRules(tx *gorm.DB, devices map[string]model.RiggingDevice) error {
	type ruleSeed struct {
		code, kind, severity, explanation string
		deviceCodes                       []string
		threshold                         map[string]any
	}
	definitions := []ruleSeed{
		{"LOAD-ALL-01", "load_limit", "blocker", "Blocks rehearsal evidence when modeled load exceeds the selected device capacity.", []string{"TRUSS-FOH-01", "LX-BRIDGE-02", "SCENIC-CLOUD-03"}, map[string]any{"use_device_limits": true}},
		{"SPEED-ALL-01", "speed_limit", "warning", "Flags modeled motion above each selected device speed limit.", []string{"TRUSS-FOH-01", "LX-BRIDGE-02", "SCENIC-CLOUD-03"}, map[string]any{"use_device_limits": true}},
		{"TRAVEL-ALL-01", "travel_limit", "blocker", "Checks modeled endpoints against each device travel envelope.", []string{"TRUSS-FOH-01", "LX-BRIDGE-02", "SCENIC-CLOUD-03"}, map[string]any{"use_device_limits": true}},
		{"ZONE-A-01", "zone_exclusion", "blocker", "Prevents simultaneous modeled motion by carriers sharing overstage-a.", []string{"TRUSS-FOH-01", "SCENIC-CLOUD-03"}, map[string]any{"minimum_gap_ms": 1000.0}},
		{"DEP-ORDER-01", "dependency_guard", "blocker", "Requires every selected cue dependency to finish before its dependent cue begins.", []string{}, map[string]any{"minimum_gap_ms": 0.0}},
	}
	for _, definition := range definitions {
		deviceIDs := make([]uint, 0, len(definition.deviceCodes))
		for _, code := range definition.deviceCodes {
			deviceIDs = append(deviceIDs, devices[code].ID)
		}
		deviceJSON, _ := json.Marshal(deviceIDs)
		thresholdJSON, _ := json.Marshal(definition.threshold)
		rule := model.InterlockRule{RuleCode: definition.code, RuleType: definition.kind, DeviceIDsJSON: datatypes.JSON(deviceJSON), ThresholdJSON: datatypes.JSON(thresholdJSON), Severity: definition.severity, Enabled: true, RuleVersion: 1, Explanation: definition.explanation}
		if err := tx.Where("rule_code = ?", rule.RuleCode).FirstOrCreate(&rule).Error; err != nil {
			return fmt.Errorf("seed rule %s: %w", rule.RuleCode, err)
		}
	}
	return nil
}

func Ping(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database handle: %w", err)
	}
	ctx, cancel := contextWithTimeout()
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	return nil
}

func contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Second)
}
