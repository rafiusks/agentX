package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/agentx/agentx-backend/internal/models"
)

// AuditLogger interface for audit logging
type AuditLogger interface {
	Log(ctx context.Context, entry *models.AuditLog) error
}

// AuditConfig holds audit middleware configuration
type AuditConfig struct {
	Logger      AuditLogger
	SkipPaths   []string // Paths to skip audit logging
	SensitiveFields []string // Fields to redact from metadata
}

// AuditMiddleware creates an audit logging middleware
func AuditMiddleware(config AuditConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if path is in skip list
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if strings.HasPrefix(path, skipPath) {
				return c.Next()
			}
		}

		// Capture start time
		startTime := time.Now()

		// Get user context if authenticated
		var userID *uuid.UUID
		if id := c.Locals("user_id"); id != nil {
			if idStr, ok := id.(string); ok {
				if parsed, err := uuid.Parse(idStr); err == nil {
					userID = &parsed
				}
			}
		}

		// Process request
		err := c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Determine action from method and path
		action := determineAction(c.Method(), path)
		
		// Determine resource type and ID from path
		resourceType, resourceID := extractResourceInfo(path)

		// Build metadata
		metadata := models.JSONB{
			"method":     c.Method(),
			"path":       path,
			"duration":   duration.Milliseconds(),
			"status":     c.Response().StatusCode(),
			"auth_method": c.Locals("auth_method"),
		}

		// Add query parameters (redact sensitive ones)
		if queryParams := c.Request().URI().QueryString(); len(queryParams) > 0 {
			metadata["query_params"] = string(queryParams)
		}

		// Determine status
		status := "success"
		var errorMessage string
		if err != nil || c.Response().StatusCode() >= 400 {
			status = "error"
			if err != nil {
				errorMessage = err.Error()
			} else {
				errorMessage = fmt.Sprintf("HTTP %d", c.Response().StatusCode())
			}
		}

		// Create audit log entry
		entry := &models.AuditLog{
			ID:           uuid.New(),
			UserID:       userID,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			IPAddress:    c.IP(),
			UserAgent:    c.Get("User-Agent"),
			Metadata:     metadata,
			Status:       status,
			ErrorMessage: errorMessage,
			CreatedAt:    time.Now(),
		}

		// Log asynchronously to avoid blocking
		go func() {
			if err := config.Logger.Log(context.Background(), entry); err != nil {
				// Log error but don't fail the request
				fmt.Printf("Failed to write audit log: %v\n", err)
			}
		}()

		return err
	}
}

// determineAction determines the action from HTTP method and path
func determineAction(method, path string) string {
	// Auth actions
	if strings.Contains(path, "/auth/login") {
		return "auth.login"
	}
	if strings.Contains(path, "/auth/logout") {
		return "auth.logout"
	}
	if strings.Contains(path, "/auth/signup") {
		return "auth.signup"
	}
	if strings.Contains(path, "/auth/refresh") {
		return "auth.refresh"
	}

	// Resource actions based on method
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 {
		resource := parts[2] // e.g., "connections", "sessions", etc.
		switch method {
		case "GET":
			if len(parts) > 3 {
				return fmt.Sprintf("%s.read", resource)
			}
			return fmt.Sprintf("%s.list", resource)
		case "POST":
			return fmt.Sprintf("%s.create", resource)
		case "PUT", "PATCH":
			return fmt.Sprintf("%s.update", resource)
		case "DELETE":
			return fmt.Sprintf("%s.delete", resource)
		}
	}

	return fmt.Sprintf("%s.%s", strings.ToLower(method), path)
}

// extractResourceInfo extracts resource type and ID from path
func extractResourceInfo(path string) (string, *uuid.UUID) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) < 3 {
		return "", nil
	}

	resourceType := parts[2] // e.g., "connections", "sessions", etc.
	
	// Try to extract ID if present
	if len(parts) > 3 {
		if id, err := uuid.Parse(parts[3]); err == nil {
			return resourceType, &id
		}
	}

	return resourceType, nil
}

// SensitiveActions that should always be logged
var SensitiveActions = []string{
	"auth.login",
	"auth.logout",
	"auth.signup",
	"connections.create",
	"connections.update",
	"connections.delete",
	"api_keys.create",
	"api_keys.delete",
	"users.update",
	"users.delete",
}

// ShouldAudit determines if an action should be audited
func ShouldAudit(action string) bool {
	for _, sensitive := range SensitiveActions {
		if action == sensitive {
			return true
		}
	}
	// Audit all write operations by default
	return strings.Contains(action, "create") ||
		strings.Contains(action, "update") ||
		strings.Contains(action, "delete")
}