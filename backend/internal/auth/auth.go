package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"stage-rigging-cue-interlock/backend/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	RoleProgrammer     = "programmer"
	RoleSafetyReviewer = "safety_reviewer"
	RoleAdmin          = "admin"
	ContextUserID      = "auth_user_id"
	ContextUsername    = "auth_username"
	ContextRole        = "auth_role"
)

type User struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:120;not null"`
	DisplayName  string `gorm:"size:120;not null"`
	Role         string `gorm:"size:32;not null;check:chk_user_role,role IN ('programmer','safety_reviewer','admin')"`
	Active       bool   `gorm:"not null;default:true"`
	CreatedAt    time.Time
}

func (User) TableName() string { return "users" }

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ByUsername(username string) (User, error) {
	var user User
	if err := r.db.Where("username = ? AND active = ?", username, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, util.Unauthorized("INVALID_CREDENTIALS", "username or password is incorrect")
		}
		return User{}, fmt.Errorf("find active user: %w", err)
	}
	return user, nil
}

func (r *Repository) ActiveByID(id uint) (User, error) {
	var user User
	if err := r.db.Where("id = ? AND active = ?", id, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return User{}, util.Unauthorized("INVALID_TOKEN", "access token user is no longer active")
		}
		return User{}, fmt.Errorf("find active user by id: %w", err)
	}
	return user, nil
}

type Service struct {
	repository *Repository
	secret     []byte
	ttl        time.Duration
}

func NewService(repository *Repository, secret string, ttl time.Duration) *Service {
	return &Service{repository: repository, secret: []byte(secret), ttl: ttl}
}

func (s *Service) Login(username, password string) (string, User, error) {
	user, err := s.repository.ByUsername(strings.TrimSpace(username))
	if err != nil {
		return "", User{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", User{}, util.Unauthorized("INVALID_CREDENTIALS", "username or password is incorrect")
	}
	now := time.Now()
	claims := Claims{UserID: user.ID, Username: user.Username, Role: user.Role, RegisteredClaims: jwt.RegisteredClaims{
		Subject: fmt.Sprint(user.ID), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", User{}, fmt.Errorf("sign access token: %w", err)
	}
	return token, user, nil
}

func (s *Service) Parse(raw string) (Claims, error) {
	claims := Claims{}
	token, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return Claims{}, util.Unauthorized("INVALID_TOKEN", "access token is invalid or expired")
	}
	user, err := s.repository.ActiveByID(claims.UserID)
	if err != nil {
		return Claims{}, err
	}
	claims.Username = user.Username
	claims.Role = user.Role
	return claims, nil
}

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Login(c *gin.Context) {
	var request struct {
		Username string `json:"username" binding:"required,min=3,max=64"`
		Password string `json:"password" binding:"required,min=8,max=128"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		util.Fail(c, util.BadRequest("VALIDATION_ERROR", "username and password are required", err.Error()))
		return
	}
	token, user, err := h.service.Login(request.Username, request.Password)
	if err != nil {
		util.Fail(c, err)
		return
	}
	util.OK(c, gin.H{"access_token": token, "token_type": "Bearer", "expires_in_seconds": int(h.service.ttl.Seconds()), "user": gin.H{"id": user.ID, "username": user.Username, "display_name": user.DisplayName, "role": user.Role}})
}

func Actor(c *gin.Context) (uint, string, string) {
	userID, _ := c.Get(ContextUserID)
	username, _ := c.Get(ContextUsername)
	role, _ := c.Get(ContextRole)
	id, _ := userID.(uint)
	name, _ := username.(string)
	actorRole, _ := role.(string)
	return id, name, actorRole
}
