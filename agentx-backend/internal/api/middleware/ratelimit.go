package middleware

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// RateLimitConfig holds rate limit configuration
type RateLimitConfig struct {
	Max        int
	Expiration time.Duration
	Message    string
}

// DefaultRateLimit returns a default rate limiter (100 requests per minute)
func DefaultRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by user ID if authenticated
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("user:%s", userID)
			}
			// Otherwise by IP
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded. Please try again later.",
			})
		},
	})
}

// AuthRateLimit returns a rate limiter for authentication endpoints (5 per minute)
func AuthRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by IP for auth endpoints
			return fmt.Sprintf("auth:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many authentication attempts. Please try again later.",
			})
		},
	})
}

// SignupRateLimit returns a rate limiter for signup endpoint (3 per hour)
func SignupRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        100, // Increased for development
		Expiration: 1 * time.Hour,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by IP for signup
			return fmt.Sprintf("signup:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many signup attempts. Please try again later.",
			})
		},
	})
}

// APIRateLimit returns a rate limiter for API endpoints (configurable)
func APIRateLimit(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by user ID if authenticated
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("api:user:%s", userID)
			}
			// Rate limit by API key if present
			if apiKeyID := c.Locals("api_key_id"); apiKeyID != nil {
				return fmt.Sprintf("api:key:%s", apiKeyID)
			}
			// Otherwise by IP
			return fmt.Sprintf("api:ip:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "API rate limit exceeded. Please slow down your requests.",
			})
		},
	})
}

// ChatRateLimit returns a rate limiter for chat endpoints (30 per minute)
func ChatRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        30,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by user ID if authenticated
			if userID := c.Locals("user_id"); userID != nil {
				return fmt.Sprintf("chat:user:%s", userID)
			}
			// Otherwise by IP
			return fmt.Sprintf("chat:ip:%s", c.IP())
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Chat rate limit exceeded. Please wait before sending more messages.",
			})
		},
		SkipSuccessfulRequests: false,
		SkipFailedRequests:     true,
	})
}