package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/models"
)

// UserRepository handles user data access
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (
			id, email, username, password_hash, full_name, 
			avatar_url, email_verified, is_active, role, settings,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Username, user.PasswordHash, user.FullName,
		user.AvatarURL, user.EmailVerified, user.IsActive, user.Role, user.Settings,
		user.CreatedAt, user.UpdatedAt,
	)
	return err
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT 
			id, email, username, password_hash, full_name,
			avatar_url, email_verified, is_active, role, settings,
			created_at, updated_at, last_login_at
		FROM users 
		WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT 
			id, email, username, password_hash, full_name,
			avatar_url, email_verified, is_active, role, settings,
			created_at, updated_at, last_login_at
		FROM users 
		WHERE email = $1`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	query := `
		SELECT 
			id, email, username, password_hash, full_name,
			avatar_url, email_verified, is_active, role, settings,
			created_at, updated_at, last_login_at
		FROM users 
		WHERE username = $1`

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users SET
			email = $2,
			username = $3,
			full_name = $4,
			avatar_url = $5,
			email_verified = $6,
			is_active = $7,
			role = $8,
			settings = $9,
			updated_at = $10
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Username, user.FullName,
		user.AvatarURL, user.EmailVerified, user.IsActive, user.Role,
		user.Settings, time.Now(),
	)
	return err
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET last_login_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, time.Now())
	return err
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, userID, passwordHash, time.Now())
	return err
}

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// List lists users with pagination
func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	var users []*models.User
	query := `
		SELECT 
			id, email, username, full_name,
			avatar_url, email_verified, is_active, role,
			created_at, updated_at, last_login_at
		FROM users 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	return users, err
}

// Count counts total users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

