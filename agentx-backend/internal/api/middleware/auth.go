package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/auth"
	"github.com/agentx/agentx-backend/internal/models"
)

// AuthConfig holds the auth middleware configuration
type AuthConfig struct {
	AuthService *auth.Service
	Optional    bool // If true, auth is optional (doesn't fail if no token)
	RequireRole string // If set, requires specific role
	RequireScope string // If set, requires specific scope (for API keys)
}

// AuthRequired creates a middleware that requires authentication
func AuthRequired(authService *auth.Service) fiber.Handler {
	return AuthMiddleware(AuthConfig{
		AuthService: authService,
		Optional:    false,
	})
}

// OptionalAuth creates a middleware that makes authentication optional
func OptionalAuth(authService *auth.Service) fiber.Handler {
	return AuthMiddleware(AuthConfig{
		AuthService: authService,
		Optional:    true,
	})
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(authService *auth.Service, role string) fiber.Handler {
	return AuthMiddleware(AuthConfig{
		AuthService: authService,
		Optional:    false,
		RequireRole: role,
	})
}

// RequireScope creates a middleware that requires a specific scope (for API keys)
func RequireScope(authService *auth.Service, scope string) fiber.Handler {
	return AuthMiddleware(AuthConfig{
		AuthService:  authService,
		Optional:     false,
		RequireScope: scope,
	})
}

// AuthMiddleware is the main authentication middleware
func AuthMiddleware(config AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to extract token from Authorization header
		authHeader := c.Get("Authorization")
		
		// Check for API key first
		if apiKey := auth.ExtractAPIKey(authHeader); apiKey != "" {
			// Validate API key
			user, key, err := config.AuthService.ValidateAPIKey(c.Context(), apiKey)
			if err != nil {
				if !config.Optional {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Invalid API key",
					})
				}
				return c.Next()
			}

			// Check scope if required
			if config.RequireScope != "" && !auth.HasScope(key.Scopes, config.RequireScope) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Insufficient scope",
				})
			}

			// Store user context
			storeUserContext(c, user, "api_key")
			c.Locals("api_key_id", key.ID.String())
			return c.Next()
		}

		// Try JWT token
		token := auth.ExtractTokenFromBearer(authHeader)
		
		// Also check for token in cookie (for web clients)
		if token == "" {
			token = c.Cookies("access_token")
		}

		// If no token and auth is optional, continue
		if token == "" && config.Optional {
			return c.Next()
		}

		// If no token and auth is required, return unauthorized
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required",
			})
		}

		// Validate token
		user, claims, err := config.AuthService.ValidateAccessToken(c.Context(), token)
		if err != nil {
			if !config.Optional {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid or expired token",
				})
			}
			return c.Next()
		}

		// Check role if required
		if config.RequireRole != "" && user.Role != config.RequireRole {
			fmt.Printf("[AUTH] Role check failed: required=%s, user=%s\n", config.RequireRole, user.Role)
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient permissions",
			})
		}

		// Store user context
		fmt.Printf("[AUTH] User authenticated: %s (role=%s)\n", user.Email, user.Role)
		storeUserContext(c, user, "jwt")
		c.Locals("session_id", claims.SessionID)

		return c.Next()
	}
}

// storeUserContext stores user information in the fiber context
func storeUserContext(c *fiber.Ctx, user *models.User, authMethod string) {
	c.Locals("user_id", user.ID.String())
	c.Locals("user_email", user.Email)
	c.Locals("user_username", user.Username)
	c.Locals("user_role", user.Role)
	c.Locals("auth_method", authMethod)
	
	// Store user context struct for easy access
	c.Locals("user_context", &models.UserContext{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	})
}

// GetUserContext retrieves the user context from the fiber context
func GetUserContext(c *fiber.Ctx) *models.UserContext {
	if ctx := c.Locals("user_context"); ctx != nil {
		if userContext, ok := ctx.(*models.UserContext); ok {
			return userContext
		}
	}
	return nil
}

// GetUserID retrieves the user ID from the fiber context
func GetUserID(c *fiber.Ctx) (uuid.UUID, error) {
	if userID := c.Locals("user_id"); userID != nil {
		if id, ok := userID.(string); ok {
			return uuid.Parse(id)
		}
	}
	return uuid.Nil, fiber.NewError(fiber.StatusUnauthorized, "User not authenticated")
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c *fiber.Ctx) bool {
	return c.Locals("user_id") != nil
}

// HasRole checks if the authenticated user has a specific role
func HasRole(c *fiber.Ctx, role string) bool {
	if userRole := c.Locals("user_role"); userRole != nil {
		if r, ok := userRole.(string); ok {
			return r == role
		}
	}
	return false
}

// IsAdmin checks if the authenticated user is an admin
func IsAdmin(c *fiber.Ctx) bool {
	return HasRole(c, models.RoleAdmin)
}

// ExtractBearerToken extracts the bearer token from the Authorization header
func ExtractBearerToken(authHeader string) string {
	parts := strings.Split(authHeader, " ")
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		return parts[1]
	}
	return ""
}