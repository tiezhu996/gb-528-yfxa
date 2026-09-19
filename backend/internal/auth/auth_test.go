package auth

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestParseUsesCurrentActiveUserAndRole(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:auth-current-user-528?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("password-528"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	user := User{Username: "programmer", PasswordHash: string(hash), DisplayName: "Programmer", Role: RoleProgrammer, Active: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	service := NewService(NewRepository(db), "test-secret", time.Hour)
	token, _, err := service.Login(user.Username, "password-528")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := db.Model(&User{}).Where("id = ?", user.ID).Updates(map[string]any{"role": RoleSafetyReviewer, "active": true}).Error; err != nil {
		t.Fatalf("change role: %v", err)
	}
	claims, err := service.Parse(token)
	if err != nil {
		t.Fatalf("parse token after role change: %v", err)
	}
	if claims.Role != RoleSafetyReviewer {
		t.Fatalf("role = %q, want current %q role", claims.Role, RoleSafetyReviewer)
	}
	if err := db.Model(&User{}).Where("id = ?", user.ID).Update("active", false).Error; err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, err := service.Parse(token); err == nil {
		t.Fatal("disabled user token remained valid")
	}
}
