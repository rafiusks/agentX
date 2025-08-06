package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Email         string     `json:"email" db:"email"`
	Username      string     `json:"username" db:"username"`
	PasswordHash  string     `json:"-" db:"password_hash"` // Never expose
	FullName      string     `json:"full_name" db:"full_name"`
	AvatarURL     string     `json:"avatar_url" db:"avatar_url"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	Role          string     `json:"role" db:"role"`
	Settings      JSONB      `json:"settings" db:"settings"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at" db:"last_login_at"`
}

// UserSession represents an active user session
type UserSession struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash        string     `json:"-" db:"token_hash"`
	RefreshTokenHash string     `json:"-" db:"refresh_token_hash"`
	ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`
	RefreshExpiresAt time.Time  `json:"refresh_expires_at" db:"refresh_expires_at"`
	IPAddress        string     `json:"ip_address" db:"ip_address"`
	UserAgent        string     `json:"user_agent" db:"user_agent"`
	DeviceName       string     `json:"device_name" db:"device_name"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	LastActivity     time.Time  `json:"last_activity" db:"last_activity"`
	RevokedAt        *time.Time `json:"revoked_at" db:"revoked_at"`
}

// UserPreferences represents user-specific preferences
type UserPreferences struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	Theme              string    `json:"theme" db:"theme"`
	DefaultModel       string    `json:"default_model" db:"default_model"`
	DefaultConnectionID *uuid.UUID `json:"default_connection_id" db:"default_connection_id"`
	UIMode             string    `json:"ui_mode" db:"ui_mode"`
	UISettings         JSONB     `json:"ui_settings" db:"ui_settings"`
	ChatSettings       JSONB     `json:"chat_settings" db:"chat_settings"`
	Shortcuts          JSONB     `json:"shortcuts" db:"shortcuts"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	Name       string     `json:"name" db:"name"`
	KeyPrefix  string     `json:"key_prefix" db:"key_prefix"`
	KeyHash    string     `json:"-" db:"key_hash"`
	LastUsedAt *time.Time `json:"last_used_at" db:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at" db:"expires_at"`
	Scopes     []string   `json:"scopes" db:"scopes"`
	IsActive   bool       `json:"is_active" db:"is_active"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	Metadata   JSONB      `json:"metadata" db:"metadata"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       *uuid.UUID `json:"user_id" db:"user_id"`
	Action       string     `json:"action" db:"action"`
	ResourceType string     `json:"resource_type" db:"resource_type"`
	ResourceID   *uuid.UUID `json:"resource_id" db:"resource_id"`
	IPAddress    string     `json:"ip_address" db:"ip_address"`
	UserAgent    string     `json:"user_agent" db:"user_agent"`
	Metadata     JSONB      `json:"metadata" db:"metadata"`
	Status       string     `json:"status" db:"status"`
	ErrorMessage string     `json:"error_message" db:"error_message"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash string     `json:"-" db:"token_hash"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt    *time.Time `json:"used_at" db:"used_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

// EmailVerificationToken represents an email verification token
type EmailVerificationToken struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash  string     `json:"-" db:"token_hash"`
	Email      string     `json:"email" db:"email"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	VerifiedAt *time.Time `json:"verified_at" db:"verified_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// UserRole constants
const (
	RoleUser    = "user"
	RoleAdmin   = "admin"
	RolePremium = "premium"
)

// UserContext represents the user context for authorization
type UserContext struct {
	UserID   uuid.UUID
	Username string
	Email    string
	Role     string
}

// IsAdmin checks if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsPremium checks if the user has premium role
func (u *User) IsPremium() bool {
	return u.Role == RolePremium
}

// JSONB type for JSON columns
type JSONB map[string]interface{}

// Value implements driver.Valuer for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
	
	return json.Unmarshal(bytes, j)
}