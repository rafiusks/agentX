package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/agentx/agentx-backend/internal/api/handlers"
	"github.com/agentx/agentx-backend/internal/services"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, svc *services.Services) {
	// API routes
	api := app.Group("/api/v1")
	
	// NOTE: Chat endpoints are in routes_auth.go as they require authentication
	// This file contains only public/admin endpoints that don't require user auth
	
	// Provider management (admin endpoints)
	api.Get("/providers", handlers.GetProviders(svc))
	api.Put("/providers/:id/config", handlers.UpdateProviderConfig(svc))
	api.Post("/providers/:id/discover", handlers.DiscoverModels(svc))
	api.Get("/providers/health", handlers.GetProvidersHealth(svc))
	api.Get("/providers/debug", handlers.DebugProviders(svc))
	
	// Session management
	api.Post("/sessions", handlers.CreateSession(svc))
	api.Get("/sessions", handlers.GetSessions(svc))
	api.Get("/sessions/:id", handlers.GetSession(svc))
	api.Delete("/sessions/:id", handlers.DeleteSession(svc))
	api.Get("/sessions/:id/messages", handlers.GetSessionMessages(svc))
	
	// Settings
	api.Get("/settings", handlers.GetSettings(svc))
	api.Put("/settings", handlers.UpdateSettings(svc))
	
	// Connections management (new multi-connection system)
	connectionHandlers := handlers.NewConnectionHandlers(svc.Connection)
	api.Get("/connections", connectionHandlers.ListConnections)
	api.Get("/connections/default", connectionHandlers.GetDefaultConnection)
	api.Post("/connections", connectionHandlers.CreateConnection)
	api.Get("/connections/:id", connectionHandlers.GetConnection)
	api.Put("/connections/:id", connectionHandlers.UpdateConnection)
	api.Delete("/connections/:id", connectionHandlers.DeleteConnection)
	api.Post("/connections/:id/toggle", connectionHandlers.ToggleConnection)
	api.Post("/connections/:id/test", connectionHandlers.TestConnection)
	api.Post("/connections/:id/set-default", connectionHandlers.SetDefaultConnection)
	
	// WebSocket routes
	// WebSocket routes are handled in routes_auth.go
	
	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"service": "agentx-backend",
		})
	})
}