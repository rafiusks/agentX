package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"

	"github.com/agentx/agentx-backend/internal/api"
	"github.com/agentx/agentx-backend/internal/config"
	"github.com/agentx/agentx-backend/internal/database"
	"github.com/agentx/agentx-backend/internal/providers"
	"github.com/agentx/agentx-backend/internal/providers/factory"
	"github.com/agentx/agentx-backend/internal/repository/postgres"
	"github.com/agentx/agentx-backend/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Connect to database
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(cfg.Database); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "AgentX Backend",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: getOrigins(),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	// Initialize repositories
	sessionRepo := postgres.NewSessionRepository(db.DB)
	messageRepo := postgres.NewMessageRepository(db.DB)
	configRepo := postgres.NewConfigRepository(db.DB)
	connectionRepo := postgres.NewConnectionRepository(db.DB)

	// Initialize provider registry
	providerRegistry := providers.NewRegistry()

	// Initialize services
	svc := services.NewServices(
		providerRegistry,
		sessionRepo,
		messageRepo,
		configRepo,
		connectionRepo,
	)

	// Initialize connections from database
	ctx := context.Background()
	if err := svc.Connection.InitializeConnections(ctx); err != nil {
		log.Printf("Failed to initialize connections: %v", err)
	} else {
		log.Println("Successfully initialized connections from database")
	}

	// Setup routes
	api.SetupRoutes(app, svc)

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Start server
	port := os.Getenv("AGENTX_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("AgentX Backend starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
		"code":  code,
	})
}

func getOrigins() string {
	origins := os.Getenv("AGENTX_CORS_ORIGINS")
	if origins == "" {
		return "http://localhost:1420,http://localhost:5173,http://localhost:3000"
	}
	return origins
}

