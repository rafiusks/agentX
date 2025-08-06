package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

var (
	// ErrInvalidScope is returned when an invalid scope is provided
	ErrInvalidScope = errors.New("invalid scope")
)

const (
	// APIKeyPrefix is the prefix for all API keys
	APIKeyPrefix = "ax_"
	// APIKeyLength is the length of the random part of the API key
	APIKeyLength = 32
)

// GenerateAPIKey generates a new API key
func GenerateAPIKey() (apiKey string, keyHash string, keyPrefix string, err error) {
	// Generate random bytes
	bytes := make([]byte, APIKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", err
	}

	// Create the API key
	randomPart := base64.URLEncoding.EncodeToString(bytes)
	// Remove padding characters
	randomPart = strings.TrimRight(randomPart, "=")
	apiKey = APIKeyPrefix + randomPart

	// Create prefix for identification (first 8 chars after prefix)
	if len(randomPart) >= 8 {
		keyPrefix = APIKeyPrefix + randomPart[:8]
	} else {
		keyPrefix = apiKey
	}

	// Hash for storage
	keyHash = HashAPIKey(apiKey)

	return apiKey, keyHash, keyPrefix, nil
}

// HashAPIKey hashes an API key for storage
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKeyFormat checks if an API key has the correct format
func ValidateAPIKeyFormat(apiKey string) bool {
	if !strings.HasPrefix(apiKey, APIKeyPrefix) {
		return false
	}
	
	// Check minimum length
	if len(apiKey) < len(APIKeyPrefix)+20 {
		return false
	}
	
	return true
}

// ExtractAPIKey extracts API key from various formats
func ExtractAPIKey(authHeader string) string {
	// Try Bearer format first
	if strings.HasPrefix(authHeader, "Bearer ") {
		key := strings.TrimPrefix(authHeader, "Bearer ")
		if ValidateAPIKeyFormat(key) {
			return key
		}
	}
	
	// Try direct API key
	if ValidateAPIKeyFormat(authHeader) {
		return authHeader
	}
	
	return ""
}

// GenerateSecureToken generates a secure random token for general use
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashToken creates a SHA256 hash of a token
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// APIKeyScopes defines available scopes for API keys
var APIKeyScopes = []string{
	"chat:read",
	"chat:write",
	"connections:read",
	"connections:write",
	"sessions:read",
	"sessions:write",
	"settings:read",
	"settings:write",
	"admin:read",
	"admin:write",
}

// ValidateScopes checks if the provided scopes are valid
func ValidateScopes(scopes []string) error {
	validScopes := make(map[string]bool)
	for _, s := range APIKeyScopes {
		validScopes[s] = true
	}
	
	for _, scope := range scopes {
		if !validScopes[scope] {
			return ErrInvalidScope
		}
	}
	
	return nil
}

// HasScope checks if a list of scopes contains a specific scope
func HasScope(scopes []string, requiredScope string) bool {
	for _, scope := range scopes {
		if scope == requiredScope {
			return true
		}
		// Check for wildcard scopes (e.g., "chat:*" matches "chat:read" and "chat:write")
		parts := strings.Split(scope, ":")
		requiredParts := strings.Split(requiredScope, ":")
		if len(parts) == 2 && len(requiredParts) == 2 {
			if parts[0] == requiredParts[0] && parts[1] == "*" {
				return true
			}
		}
	}
	return false
}