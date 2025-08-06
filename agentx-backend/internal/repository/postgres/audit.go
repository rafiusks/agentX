package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/models"
)

// AuditLogRepository handles audit log data access
type AuditLogRepository struct {
	db *sqlx.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Log creates a new audit log entry
func (r *AuditLogRepository) Log(ctx context.Context, entry *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			id, user_id, action, resource_type, resource_id,
			ip_address, user_agent, metadata, status, error_message,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.db.ExecContext(ctx, query,
		entry.ID, entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID,
		entry.IPAddress, entry.UserAgent, entry.Metadata, entry.Status, entry.ErrorMessage,
		entry.CreatedAt,
	)
	return err
}

// GetByID retrieves an audit log entry by ID
func (r *AuditLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var entry models.AuditLog
	query := `SELECT * FROM audit_logs WHERE id = $1`

	err := r.db.GetContext(ctx, &entry, query, id)
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetByUserID lists audit logs for a specific user
func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*models.AuditLog, error) {
	var entries []*models.AuditLog
	query := `
		SELECT * FROM audit_logs 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &entries, query, userID, limit)
	return entries, err
}

// GetRecentLogs gets recent audit logs
func (r *AuditLogRepository) GetRecentLogs(ctx context.Context, limit int) ([]*models.AuditLog, error) {
	var entries []*models.AuditLog
	query := `
		SELECT * FROM audit_logs 
		ORDER BY created_at DESC
		LIMIT $1`

	err := r.db.SelectContext(ctx, &entries, query, limit)
	return entries, err
}

// ListByUser lists audit logs for a specific user
func (r *AuditLogRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.AuditLog, error) {
	var entries []*models.AuditLog
	query := `
		SELECT * FROM audit_logs 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	err := r.db.SelectContext(ctx, &entries, query, userID, limit, offset)
	return entries, err
}

// ListByAction lists audit logs for a specific action
func (r *AuditLogRepository) ListByAction(ctx context.Context, action string, limit, offset int) ([]*models.AuditLog, error) {
	var entries []*models.AuditLog
	query := `
		SELECT * FROM audit_logs 
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	err := r.db.SelectContext(ctx, &entries, query, action, limit, offset)
	return entries, err
}

// ListByResource lists audit logs for a specific resource
func (r *AuditLogRepository) ListByResource(ctx context.Context, resourceType string, resourceID uuid.UUID, limit, offset int) ([]*models.AuditLog, error) {
	var entries []*models.AuditLog
	query := `
		SELECT * FROM audit_logs 
		WHERE resource_type = $1 AND resource_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	err := r.db.SelectContext(ctx, &entries, query, resourceType, resourceID, limit, offset)
	return entries, err
}

// List lists all audit logs with filters
func (r *AuditLogRepository) List(ctx context.Context, filter AuditLogFilter) ([]*models.AuditLog, error) {
	var entries []*models.AuditLog
	query := `
		SELECT * FROM audit_logs 
		WHERE 1=1`
	
	var args []interface{}
	argCount := 0

	// Build dynamic query based on filters
	if filter.UserID != nil {
		argCount++
		query += ` AND user_id = $` + string(rune(argCount))
		args = append(args, *filter.UserID)
	}

	if filter.Action != "" {
		argCount++
		query += ` AND action = $` + string(rune(argCount))
		args = append(args, filter.Action)
	}

	if filter.ResourceType != "" {
		argCount++
		query += ` AND resource_type = $` + string(rune(argCount))
		args = append(args, filter.ResourceType)
	}

	if filter.Status != "" {
		argCount++
		query += ` AND status = $` + string(rune(argCount))
		args = append(args, filter.Status)
	}

	if !filter.StartDate.IsZero() {
		argCount++
		query += ` AND created_at >= $` + string(rune(argCount))
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		argCount++
		query += ` AND created_at <= $` + string(rune(argCount))
		args = append(args, filter.EndDate)
	}

	query += ` ORDER BY created_at DESC`

	if filter.Limit > 0 {
		argCount++
		query += ` LIMIT $` + string(rune(argCount))
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += ` OFFSET $` + string(rune(argCount))
		args = append(args, filter.Offset)
	}

	err := r.db.SelectContext(ctx, &entries, query, args...)
	return entries, err
}

// Count counts audit logs matching filter
func (r *AuditLogRepository) Count(ctx context.Context, filter AuditLogFilter) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM audit_logs 
		WHERE 1=1`
	
	var args []interface{}
	argCount := 0

	if filter.UserID != nil {
		argCount++
		query += ` AND user_id = $` + string(rune(argCount))
		args = append(args, *filter.UserID)
	}

	if filter.Action != "" {
		argCount++
		query += ` AND action = $` + string(rune(argCount))
		args = append(args, filter.Action)
	}

	if filter.ResourceType != "" {
		argCount++
		query += ` AND resource_type = $` + string(rune(argCount))
		args = append(args, filter.ResourceType)
	}

	if filter.Status != "" {
		argCount++
		query += ` AND status = $` + string(rune(argCount))
		args = append(args, filter.Status)
	}

	if !filter.StartDate.IsZero() {
		argCount++
		query += ` AND created_at >= $` + string(rune(argCount))
		args = append(args, filter.StartDate)
	}

	if !filter.EndDate.IsZero() {
		argCount++
		query += ` AND created_at <= $` + string(rune(argCount))
		args = append(args, filter.EndDate)
	}

	err := r.db.GetContext(ctx, &count, query, args...)
	return count, err
}

// DeleteOlderThan deletes audit logs older than specified date
func (r *AuditLogRepository) DeleteOlderThan(ctx context.Context, date time.Time) error {
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	_, err := r.db.ExecContext(ctx, query, date)
	return err
}

// AuditLogFilter represents filters for audit log queries
type AuditLogFilter struct {
	UserID       *uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	Status       string
	StartDate    time.Time
	EndDate      time.Time
	Limit        int
	Offset       int
}

// GetRecentFailedLogins gets recent failed login attempts
func (r *AuditLogRepository) GetRecentFailedLogins(ctx context.Context, ipAddress string, duration time.Duration) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM audit_logs 
		WHERE action = 'auth.login' 
		AND status = 'error'
		AND ip_address = $1
		AND created_at > $2`

	since := time.Now().Add(-duration)
	err := r.db.GetContext(ctx, &count, query, ipAddress, since)
	return count, err
}

// GetUserActivity gets a summary of user activity
func (r *AuditLogRepository) GetUserActivity(ctx context.Context, userID uuid.UUID, days int) (map[string]int, error) {
	type ActivityCount struct {
		Action string `db:"action"`
		Count  int    `db:"count"`
	}

	var activities []ActivityCount
	query := `
		SELECT action, COUNT(*) as count
		FROM audit_logs 
		WHERE user_id = $1
		AND created_at > $2
		GROUP BY action
		ORDER BY count DESC`

	since := time.Now().AddDate(0, 0, -days)
	err := r.db.SelectContext(ctx, &activities, query, userID, since)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, activity := range activities {
		result[activity.Action] = activity.Count
	}

	return result, nil
}