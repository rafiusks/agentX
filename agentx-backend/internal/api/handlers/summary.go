package handlers

import (
	"github.com/agentx/agentx-backend/internal/models"
	"github.com/agentx/agentx-backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

type SummaryHandler struct {
	summaryService *services.SummaryService
}

func NewSummaryHandler(summaryService *services.SummaryService) *SummaryHandler {
	return &SummaryHandler{
		summaryService: summaryService,
	}
}

// GenerateSummary handles POST /api/v1/sessions/:id/summary
func (h *SummaryHandler) GenerateSummary(c *fiber.Ctx) error {
	// Get user context
	userContext, ok := c.Locals("user").(*models.UserContext)
	if !ok || userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Parse request body
	var req struct {
		MessageCount int `json:"message_count"`
	}
	if err := c.BodyParser(&req); err != nil {
		req.MessageCount = 20 // Default to 20 messages
	}

	// Generate summary
	summary, err := h.summaryService.GenerateSessionSummary(
		c.Context(),
		userContext.UserID.String(),
		sessionID,
		req.MessageCount,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate summary",
			"details": err.Error(),
		})
	}

	return c.JSON(summary)
}

// GetSummaries handles GET /api/v1/sessions/:id/summaries
func (h *SummaryHandler) GetSummaries(c *fiber.Ctx) error {
	// Get user context
	userContext, ok := c.Locals("user").(*models.UserContext)
	if !ok || userContext == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Session ID is required",
		})
	}

	// Get summaries
	summaries, err := h.summaryService.GetSessionSummaries(
		c.Context(),
		userContext.UserID.String(),
		sessionID,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch summaries",
			"details": err.Error(),
		})
	}

	return c.JSON(summaries)
}