package handlers

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/api/middleware"
	"github.com/agentx/agentx-backend/internal/audit"
	"github.com/agentx/agentx-backend/internal/auth"
	"github.com/agentx/agentx-backend/internal/services"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	DeviceName string `json:"device_name"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token"`
	RefreshToken string        `json:"refresh_token"`
	ExpiresIn    int           `json:"expires_in"`
}

// SignupRequest represents a signup request
type SignupRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshResponse represents a token refresh response
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// generateRandomSuffix generates a random 4-character suffix
func generateRandomSuffix() string {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 4)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	FullName      string    `json:"full_name"`
	AvatarURL     string    `json:"avatar_url"`
	EmailVerified bool      `json:"email_verified"`
	Role          string    `json:"role"`
	CreatedAt     time.Time `json:"created_at"`
}

// Login handles user login
func Login(authService *auth.Service, auditService *audit.Service, svc *services.Services) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate request
		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Email and password are required",
			})
		}

		// Get device info
		ipAddress := c.IP()
		userAgent := c.Get("User-Agent")
		deviceName := req.DeviceName
		if deviceName == "" {
			deviceName = "Unknown Device"
		}

		// Attempt login
		user, accessToken, refreshToken, err := authService.Login(
			c.Context(),
			req.Email,
			req.Password,
			ipAddress,
			userAgent,
			deviceName,
		)
		if err != nil {
			// Log the actual error for debugging
			fmt.Printf("[Login Error] %v\n", err)
			
			// Don't reveal specific error to prevent user enumeration
			if err == auth.ErrInvalidCredentials || err == auth.ErrUserNotFound {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid email or password",
				})
			}
			if err == auth.ErrUserInactive {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Account is inactive",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Login failed",
			})
		}

		// Log successful login
		event := audit.NewEvent(audit.EventLogin, &user.ID, c.IP(), c.Get("User-Agent"))
		event.Resource = "auth"
		event.Action = "login"
		event.Result = "success"
		event.Metadata["email"] = user.Email
		auditService.Log(c.Context(), event)

		// Initialize user connections after successful login through Orchestrator
		if svc != nil && svc.Orchestrator != nil {
			if err := svc.Orchestrator.InitializeUserConnections(c.Context(), user.ID); err != nil {
				// Log error but don't fail login
				c.App().Config().ErrorHandler(c, err)
			}
		}

		// Set cookies for web clients
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    accessToken,
			Expires:  time.Now().Add(auth.AccessTokenTTL),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Expires:  time.Now().Add(auth.RefreshTokenTTL),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		return c.JSON(LoginResponse{
			User: &UserResponse{
				ID:            user.ID.String(),
				Email:         user.Email,
				Username:      user.Username,
				FullName:      user.FullName,
				AvatarURL:     user.AvatarURL,
				EmailVerified: user.EmailVerified,
				Role:          user.Role,
				CreatedAt:     user.CreatedAt,
			},
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int(auth.AccessTokenTTL.Seconds()),
		})
	}
}

// Signup handles user registration
func Signup(authService *auth.Service, auditService *audit.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req SignupRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate password strength
		if err := auth.ValidatePassword(req.Password); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Use email as username (extract the part before @)
		username := req.Email
		if atIndex := strings.Index(req.Email, "@"); atIndex > 0 {
			username = req.Email[:atIndex]
		}

		// Create user
		user, err := authService.SignUp(
			c.Context(),
			req.Email,
			username,
			req.Password,
			req.FullName,
		)
		if err != nil {
			if err == auth.ErrEmailAlreadyExists {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error": "Email already registered",
				})
			}
			if err == auth.ErrUsernameAlreadyExists {
				// If username conflicts, try with a random suffix
				username = username + "_" + generateRandomSuffix()
				user, err = authService.SignUp(
					c.Context(),
					req.Email,
					username,
					req.Password,
					req.FullName,
				)
				if err != nil {
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error": "Registration failed",
					})
				}
			} else {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Registration failed",
				})
			}
		}

		// Auto-login after signup
		ipAddress := c.IP()
		userAgent := c.Get("User-Agent")
		_, accessToken, refreshToken, err := authService.Login(
			c.Context(),
			req.Email,
			req.Password,
			ipAddress,
			userAgent,
			"New Registration",
		)
		if err != nil {
			// User created but login failed - they can login manually
			return c.JSON(fiber.Map{
				"user": &UserResponse{
					ID:            user.ID.String(),
					Email:         user.Email,
					Username:      user.Username,
					FullName:      user.FullName,
					EmailVerified: user.EmailVerified,
					Role:          user.Role,
					CreatedAt:     user.CreatedAt,
				},
				"message": "Registration successful. Please login.",
			})
		}

		return c.JSON(LoginResponse{
			User: &UserResponse{
				ID:            user.ID.String(),
				Email:         user.Email,
				Username:      user.Username,
				FullName:      user.FullName,
				EmailVerified: user.EmailVerified,
				Role:          user.Role,
				CreatedAt:     user.CreatedAt,
			},
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int(auth.AccessTokenTTL.Seconds()),
		})
	}
}

// RefreshToken handles token refresh
func RefreshToken(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get refresh token from body first
		var req RefreshRequest
		c.BodyParser(&req)
		
		refreshToken := req.RefreshToken
		
		// If not in body, try Authorization header
		if refreshToken == "" {
			refreshToken = auth.ExtractTokenFromBearer(c.Get("Authorization"))
		}
		
		// If not in header, try cookie
		if refreshToken == "" {
			refreshToken = c.Cookies("refresh_token")
		}

		if refreshToken == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Refresh token required",
			})
		}

		// Refresh tokens
		newAccessToken, newRefreshToken, err := authService.RefreshToken(c.Context(), refreshToken)
		if err != nil {
			if err == auth.ErrInvalidToken || err == auth.ErrExpiredToken {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid or expired refresh token",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Token refresh failed",
			})
		}

		// Update cookies
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    newAccessToken,
			Expires:  time.Now().Add(auth.AccessTokenTTL),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    newRefreshToken,
			Expires:  time.Now().Add(auth.RefreshTokenTTL),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		return c.JSON(RefreshResponse{
			AccessToken:  newAccessToken,
			RefreshToken: newRefreshToken,
			ExpiresIn:    int(auth.AccessTokenTTL.Seconds()),
		})
	}
}

// Logout handles user logout
func Logout(authService *auth.Service, auditService *audit.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get session ID from context
		sessionID := c.Locals("session_id")
		if sessionID != nil {
			if id, ok := sessionID.(string); ok {
				// Revoke the session
				if err := authService.Logout(c.Context(), id); err != nil {
					// Log error but don't fail logout
					c.App().Config().ErrorHandler(c, err)
				}
			}
		}

		// Clear cookies
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    "",
			Expires:  time.Now().Add(-time.Hour),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Expires:  time.Now().Add(-time.Hour),
			HTTPOnly: true,
			Secure:   true,
			SameSite: "Strict",
		})

		return c.JSON(fiber.Map{
			"message": "Logged out successfully",
		})
	}
}

// GetCurrentUser returns the current authenticated user
func GetCurrentUser(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		// Get fresh user data
		user, err := authService.GetUser(c.Context(), userContext.UserID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to get user data",
			})
		}

		return c.JSON(&UserResponse{
			ID:            user.ID.String(),
			Email:         user.Email,
			Username:      user.Username,
			FullName:      user.FullName,
			AvatarURL:     user.AvatarURL,
			EmailVerified: user.EmailVerified,
			Role:          user.Role,
			CreatedAt:     user.CreatedAt,
		})
	}
}

// UpdateProfile updates the current user's profile
func UpdateProfile(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		var req struct {
			FullName  string `json:"full_name"`
			AvatarURL string `json:"avatar_url"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Update user profile
		user, err := authService.UpdateProfile(
			c.Context(),
			userContext.UserID,
			req.FullName,
			req.AvatarURL,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to update profile",
			})
		}

		return c.JSON(&UserResponse{
			ID:            user.ID.String(),
			Email:         user.Email,
			Username:      user.Username,
			FullName:      user.FullName,
			AvatarURL:     user.AvatarURL,
			EmailVerified: user.EmailVerified,
			Role:          user.Role,
			CreatedAt:     user.CreatedAt,
		})
	}
}

// ChangePassword changes the current user's password
func ChangePassword(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userContext := middleware.GetUserContext(c)
		if userContext == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}

		var req struct {
			CurrentPassword string `json:"current_password" validate:"required"`
			NewPassword     string `json:"new_password" validate:"required,min=8"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		// Validate new password
		if err := auth.ValidatePassword(req.NewPassword); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		// Change password
		if err := authService.ChangePassword(
			c.Context(),
			userContext.UserID,
			req.CurrentPassword,
			req.NewPassword,
		); err != nil {
			if err == auth.ErrInvalidCredentials {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Current password is incorrect",
				})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to change password",
			})
		}

		return c.JSON(fiber.Map{
			"message": "Password changed successfully",
		})
	}
}