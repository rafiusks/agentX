package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// AccessTokenTTL is the access token time to live
	AccessTokenTTL = 15 * time.Minute
	// RefreshTokenTTL is the refresh token time to live
	RefreshTokenTTL = 7 * 24 * time.Hour
)

var (
	// ErrInvalidToken is returned when a token is invalid
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken is returned when a token is expired
	ErrExpiredToken = errors.New("token expired")
	// ErrInvalidClaims is returned when token claims are invalid
	ErrInvalidClaims = errors.New("invalid token claims")
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID    string   `json:"user_id"`
	Email     string   `json:"email"`
	Username  string   `json:"username"`
	Role      string   `json:"role"`
	SessionID string   `json:"session_id"`
	TokenType string   `json:"token_type"` // "access" or "refresh"
	Scopes    []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

// JWTService handles JWT operations
type JWTService struct {
	secretKey []byte
	issuer    string
}

// NewJWTService creates a new JWT service
func NewJWTService(secretKey string, issuer string) *JWTService {
	return &JWTService{
		secretKey: []byte(secretKey),
		issuer:    issuer,
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (s *JWTService) GenerateTokenPair(userID, email, username, role string, sessionID string) (accessToken string, refreshToken string, err error) {
	// Generate access token
	accessToken, err = s.GenerateAccessToken(userID, email, username, role, sessionID)
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshToken, err = s.GenerateRefreshToken(userID, sessionID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateAccessToken generates an access token
func (s *JWTService) GenerateAccessToken(userID, email, username, role string, sessionID string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		Username:  username,
		Role:      role,
		SessionID: sessionID,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// GenerateRefreshToken generates a refresh token
func (s *JWTService) GenerateRefreshToken(userID string, sessionID string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenTTL)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secretKey)
}

// ValidateToken validates a token and returns the claims
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

// ValidateAccessToken validates an access token
func (s *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token
func (s *JWTService) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ExtractTokenFromBearer extracts token from "Bearer <token>" format
func ExtractTokenFromBearer(authHeader string) string {
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:]
	}
	return ""
}