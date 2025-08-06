package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/agentx/agentx-backend/internal/api/handlers"
	"github.com/agentx/agentx-backend/internal/services"
)

// SetupRoutes configures all API routes
func SetupRoutes(app *fiber.App, svc *services.Services) {
	// API routes
	api := app.Group("/api/v1")
	
	// Create unified chat handler
	unifiedHandler := handlers.NewUnifiedChatHandler(svc.UnifiedChat)
	
	// Unified endpoints (new, frontend-agnostic)
	api.Post("/chat", unifiedHandler.Chat)
	api.Post("/chat/stream", unifiedHandler.StreamChatSSE)  // SSE endpoint
	api.Get("/models", unifiedHandler.GetModels)
	
	// Legacy endpoints for backward compatibility
	api.Post("/chat/completions", unifiedHandler.ChatCompletions)  // OpenAI-compatible
	
	// Provider management (admin endpoints)
	api.Get("/providers", handlers.GetProviders(svc))
	api.Put("/providers/:id/config", handlers.UpdateProviderConfig(svc))
	api.Post("/providers/:id/discover", handlers.DiscoverModels(svc))
	api.Get("/providers/health", handlers.GetProvidersHealth(svc))
	
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
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	
	app.Get("/ws/chat", websocket.New(unifiedHandler.StreamChat))
	
	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		// Get health status from all providers
		healthMonitor := svc.Router.GetHealthMonitor()
		health := healthMonitor.GetAllHealth()
		
		// Determine overall health
		overallHealthy := true
		for _, status := range health {
			if !status.Healthy {
				overallHealthy = false
				break
			}
		}
		
		status := "healthy"
		if !overallHealthy {
			status = "degraded"
		}
		
		return c.JSON(fiber.Map{
			"status": status,
			"service": "agentx-backend",
			"providers": health,
		})
	})
}