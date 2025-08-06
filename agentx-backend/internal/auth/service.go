package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/models"
)

var (
	// ErrInvalidCredentials is returned when login credentials are invalid
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")
	// ErrUserInactive is returned when a user is inactive
	ErrUserInactive = errors.New("user account is inactive")
	// ErrEmailAlreadyExists is returned when email is already registered
	ErrEmailAlreadyExists = errors.New("email already exists")
	// ErrUsernameAlreadyExists is returned when username is already taken
	ErrUsernameAlreadyExists = errors.New("username already exists")
	// ErrSessionNotFound is returned when a session is not found
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired is returned when a session is expired
	ErrSessionExpired = errors.New("session expired")
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// SessionRepository defines the interface for session data access
type SessionRepository interface {
	Create(ctx context.Context, session *models.UserSession) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.UserSession, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*models.UserSession, error)
	GetByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*models.UserSession, error)
	Update(ctx context.Context, session *models.UserSession) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) error
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
}

// APIKeyRepository defines the interface for API key data access
type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *models.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error)
	GetByKeyHash(ctx context.Context, keyHash string) (*models.APIKey, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*models.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service handles authentication operations
type Service struct {
	userRepo   UserRepository
	sessionRepo SessionRepository
	apiKeyRepo APIKeyRepository
	jwt        *JWTService
}

// NewService creates a new auth service
func NewService(userRepo UserRepository, sessionRepo SessionRepository, apiKeyRepo APIKeyRepository, jwtSecret string) *Service {
	return &Service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		apiKeyRepo:  apiKeyRepo,
		jwt:         NewJWTService(jwtSecret, "agentx"),
	}
}

// SignUp registers a new user
func (s *Service) SignUp(ctx context.Context, email, username, password, fullName string) (*models.User, error) {
	// Validate password
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	// Check if email exists
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailAlreadyExists
	}

	// Check if username exists
	existingUser, err = s.userRepo.GetByUsername(ctx, username)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrUsernameAlreadyExists
	}

	// Hash password
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		FullName:     fullName,
		IsActive:     true,
		Role:         models.RoleUser,
		Settings:     make(models.JSONB),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user and creates a session
func (s *Service) Login(ctx context.Context, email, password string, ipAddress, userAgent, deviceName string) (*models.User, string, string, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", "", ErrInvalidCredentials
		}
		return nil, "", "", err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, "", "", ErrUserInactive
	}

	// Verify password
	if !CheckPassword(password, user.PasswordHash) {
		return nil, "", "", ErrInvalidCredentials
	}

	// Create session
	session := &models.UserSession{
		ID:               uuid.New(),
		UserID:           user.ID,
		ExpiresAt:        time.Now().Add(AccessTokenTTL),
		RefreshExpiresAt: time.Now().Add(RefreshTokenTTL),
		IPAddress:        ipAddress,
		UserAgent:        userAgent,
		DeviceName:       deviceName,
		CreatedAt:        time.Now(),
		LastActivity:     time.Now(),
	}

	// Generate tokens
	accessToken, refreshToken, err := s.jwt.GenerateTokenPair(
		user.ID.String(),
		user.Email,
		user.Username,
		user.Role,
		session.ID.String(),
	)
	if err != nil {
		return nil, "", "", err
	}

	// Hash tokens for storage
	session.TokenHash = HashToken(accessToken)
	session.RefreshTokenHash = HashToken(refreshToken)

	// Save session
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, "", "", err
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	return user, accessToken, refreshToken, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// Validate refresh token
	claims, err := s.jwt.ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	// Get session
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return "", "", ErrInvalidToken
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", ErrSessionNotFound
		}
		return "", "", err
	}

	// Check if session is revoked
	if session.RevokedAt != nil {
		return "", "", ErrSessionExpired
	}

	// Verify refresh token hash
	if session.RefreshTokenHash != HashToken(refreshToken) {
		return "", "", ErrInvalidToken
	}

	// Check if refresh token is expired
	if session.RefreshExpiresAt.Before(time.Now()) {
		return "", "", ErrSessionExpired
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return "", "", err
	}

	// Generate new tokens
	newAccessToken, newRefreshToken, err := s.jwt.GenerateTokenPair(
		user.ID.String(),
		user.Email,
		user.Username,
		user.Role,
		session.ID.String(),
	)
	if err != nil {
		return "", "", err
	}

	// Update session with new token hashes and expiry
	session.TokenHash = HashToken(newAccessToken)
	session.RefreshTokenHash = HashToken(newRefreshToken)
	session.ExpiresAt = time.Now().Add(AccessTokenTTL)
	session.RefreshExpiresAt = time.Now().Add(RefreshTokenTTL)
	session.LastActivity = time.Now()

	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

// Logout revokes a session
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return err
	}

	session, err := s.sessionRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	now := time.Now()
	session.RevokedAt = &now
	return s.sessionRepo.Update(ctx, session)
}

// ValidateAccessToken validates an access token and returns the user
func (s *Service) ValidateAccessToken(ctx context.Context, token string) (*models.User, *JWTClaims, error) {
	claims, err := s.jwt.ValidateAccessToken(token)
	if err != nil {
		return nil, nil, err
	}

	// Get session to verify it's still valid
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrSessionNotFound
		}
		return nil, nil, err
	}

	// Check if session is revoked
	if session.RevokedAt != nil {
		return nil, nil, ErrSessionExpired
	}

	// Get user
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, nil, ErrInvalidToken
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	// Update last activity
	session.LastActivity = time.Now()
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to update session activity: %v\n", err)
	}

	return user, claims, nil
}

// ValidateAPIKey validates an API key and returns the user
func (s *Service) ValidateAPIKey(ctx context.Context, apiKey string) (*models.User, *models.APIKey, error) {
	if !ValidateAPIKeyFormat(apiKey) {
		return nil, nil, ErrInvalidToken
	}

	keyHash := HashAPIKey(apiKey)
	key, err := s.apiKeyRepo.GetByKeyHash(ctx, keyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrInvalidToken
		}
		return nil, nil, err
	}

	// Check if key is active
	if !key.IsActive {
		return nil, nil, ErrInvalidToken
	}

	// Check if key is expired
	if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
		return nil, nil, ErrExpiredToken
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, key.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, nil, ErrUserInactive
	}

	// Update last used
	if err := s.apiKeyRepo.UpdateLastUsed(ctx, key.ID); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to update API key last used: %v\n", err)
	}

	return user, key, nil
}

// CreateAPIKey creates a new API key for a user
func (s *Service) CreateAPIKey(ctx context.Context, userID uuid.UUID, name string, scopes []string, expiresAt *time.Time) (*models.APIKey, string, error) {
	// Validate scopes
	if err := ValidateScopes(scopes); err != nil {
		return nil, "", err
	}

	// Generate API key
	apiKey, keyHash, keyPrefix, err := GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}

	// Create API key record
	key := &models.APIKey{
		ID:        uuid.New(),
		UserID:    userID,
		Name:      name,
		KeyPrefix: keyPrefix,
		KeyHash:   keyHash,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		IsActive:  true,
		CreatedAt: time.Now(),
		Metadata:  make(models.JSONB),
	}

	if err := s.apiKeyRepo.Create(ctx, key); err != nil {
		return nil, "", err
	}

	return key, apiKey, nil
}

// RevokeAPIKey revokes an API key
func (s *Service) RevokeAPIKey(ctx context.Context, userID, keyID uuid.UUID) error {
	// Verify the key belongs to the user
	key, err := s.apiKeyRepo.GetByID(ctx, keyID)
	if err != nil {
		return err
	}

	if key.UserID != userID {
		return errors.New("unauthorized")
	}

	return s.apiKeyRepo.Delete(ctx, keyID)
}

// CleanupExpiredSessions removes expired sessions
func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	return s.sessionRepo.DeleteExpired(ctx)
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*models.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// UpdateProfile updates a user's profile
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, fullName, avatarURL string) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	user.FullName = fullName
	user.AvatarURL = avatarURL
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !CheckPassword(currentPassword, user.PasswordHash) {
		return ErrInvalidCredentials
	}

	// Validate new password
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	// Hash new password
	newHash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	// Update password
	return s.userRepo.UpdatePassword(ctx, userID, newHash)
}

// API Key Management

// GetAPIKeysByUserID retrieves all API keys for a user
func (s *Service) GetAPIKeysByUserID(ctx context.Context, userID uuid.UUID) ([]*models.APIKey, error) {
	return s.apiKeyRepo.ListByUserID(ctx, userID)
}