package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/agentx/agentx-backend/internal/api/handlers"
	"github.com/agentx/agentx-backend/internal/api/middleware"
	"github.com/agentx/agentx-backend/internal/audit"
	"github.com/agentx/agentx-backend/internal/auth"
	"github.com/agentx/agentx-backend/internal/services"
)

// SetupRoutesWithAuth configures all API routes with authentication
func SetupRoutesWithAuth(app *fiber.App, svc *services.Services, authService *auth.Service, auditService *audit.Service) {
	// API routes
	api := app.Group("/api/v1")
	
	// ========================================
	// Public routes (no authentication needed)
	// ========================================
	
	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"service": "agentx-backend",
		})
	})
	
	// Authentication endpoints
	auth := api.Group("/auth")
	auth.Post("/login", middleware.AuthRateLimit(), handlers.Login(authService, auditService, svc))
	auth.Post("/signup", middleware.SignupRateLimit(), handlers.Signup(authService, auditService))
	auth.Post("/refresh", handlers.RefreshToken(authService))
	auth.Post("/logout", middleware.AuthRequired(authService), handlers.Logout(authService, auditService))
	
	// ========================================
	// Protected routes (authentication required)
	// ========================================
	
	// Apply auth middleware to all protected routes
	protected := api.Group("", middleware.AuthRequired(authService))
	
	// User profile endpoints  
	protected.Get("/auth/me", handlers.GetCurrentUser(authService))
	protected.Put("/auth/profile", handlers.UpdateProfile(authService))
	protected.Put("/auth/password", handlers.ChangePassword(authService))
	
	// Create unified chat handler using the Orchestrator
	unifiedHandler := handlers.NewUnifiedChatHandler(svc.Orchestrator)
	
	// Chat endpoints
	protected.Post("/chat", unifiedHandler.Chat)
	protected.Post("/chat/stream", unifiedHandler.StreamChatSSE)  // SSE endpoint
	protected.Get("/models", unifiedHandler.GetModels)
	
	// Legacy endpoints for backward compatibility
	protected.Post("/chat/completions", unifiedHandler.ChatCompletions)  // OpenAI-compatible
	
	// Session management
	protected.Post("/sessions", handlers.CreateSession(svc))
	protected.Get("/sessions", handlers.GetSessions(svc))
	protected.Get("/sessions/:id", handlers.GetSession(svc))
	protected.Put("/sessions/:id", handlers.UpdateSession(svc))
	protected.Delete("/sessions/:id", handlers.DeleteSession(svc))
	protected.Get("/sessions/:id/messages", handlers.GetSessionMessages(svc))
	
	// LLM service (NEW: General AI operations)
	llmHandler := handlers.NewLLMHandler(svc.LLM)
	llmHandler.RegisterRoutes(protected)
	
	// Connections management (multi-connection system)
	connectionHandlers := handlers.NewConnectionHandlers(svc.Connection)
	protected.Get("/connections", connectionHandlers.ListConnections)
	protected.Get("/connections/default", connectionHandlers.GetDefaultConnection)
	protected.Post("/connections", connectionHandlers.CreateConnection)
	protected.Get("/connections/:id", connectionHandlers.GetConnection)
	protected.Put("/connections/:id", connectionHandlers.UpdateConnection)
	protected.Delete("/connections/:id", connectionHandlers.DeleteConnection)
	protected.Post("/connections/:id/toggle", connectionHandlers.ToggleConnection)
	protected.Post("/connections/:id/test", connectionHandlers.TestConnection)
	protected.Post("/connections/:id/set-default", connectionHandlers.SetDefaultConnection)
	
	// Provider management (admin only)
	admin := protected.Group("", middleware.RequireRole(authService, "admin"))
	admin.Get("/providers", handlers.GetProviders(svc))
	admin.Put("/providers/:id/config", handlers.UpdateProviderConfig(svc))
	admin.Post("/providers/:id/discover", handlers.DiscoverModels(svc))
	admin.Get("/providers/health", handlers.GetProvidersHealth(svc))
	
	// Settings (user-specific)
	protected.Get("/settings", handlers.GetSettings(svc))
	protected.Put("/settings", handlers.UpdateSettings(svc))
	
	// Context Memory management
	contextHandlers := handlers.NewContextMemoryHandlers(svc.ContextMemory)
	protected.Post("/context/memory", contextHandlers.StoreMemory)
	protected.Get("/context/memory", contextHandlers.ListMemories)
	protected.Get("/context/memory/search", contextHandlers.SearchMemories)
	protected.Get("/context/memory/relevant/:sessionId", contextHandlers.GetRelevantMemories)
	protected.Get("/context/memory/:namespace/:key", contextHandlers.GetMemory)
	protected.Delete("/context/memory/:namespace/:key", contextHandlers.DeleteMemory)
	protected.Put("/context/memory/:id/importance", contextHandlers.UpdateImportance)
	
	// API Key management
	protected.Get("/api-keys", handlers.ListAPIKeys(authService))
	protected.Post("/api-keys", handlers.CreateAPIKey(authService))
	protected.Delete("/api-keys/:id", handlers.RevokeAPIKey(authService))
	
	// ========================================
	// WebSocket routes (with auth)
	// ========================================
	
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			// Validate auth token from query param or header
			token := c.Query("token")
			if token == "" {
				token = c.Get("Authorization")
				if len(token) > 7 && token[:7] == "Bearer " {
					token = token[7:]
				}
			}
			
			if token != "" {
				user, claims, err := authService.ValidateAccessToken(c.Context(), token)
				if err == nil {
					c.Locals("user", user)
					c.Locals("claims", claims)
					c.Locals("allowed", true)
					return c.Next()
				}
			}
			
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required for WebSocket",
			})
		}
		return fiber.ErrUpgradeRequired
	})
	
	app.Get("/ws/chat", websocket.New(unifiedHandler.StreamChat))
}