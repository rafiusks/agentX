package services

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/agentx/agentx-backend/internal/repository"
)

// DatabaseService provides unified database access
// This will eventually wrap all repository operations
type DatabaseService struct {
	db          *sqlx.DB
	sessionRepo repository.SessionRepository
	messageRepo repository.MessageRepository
	configRepo  repository.ConfigRepository
	connRepo    repository.ConnectionRepository
}

// NewDatabaseService creates a new database service
func NewDatabaseService(
	db *sqlx.DB,
	sessionRepo repository.SessionRepository,
	messageRepo repository.MessageRepository,
	configRepo repository.ConfigRepository,
	connRepo repository.ConnectionRepository,
) *DatabaseService {
	return &DatabaseService{
		db:          db,
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		configRepo:  configRepo,
		connRepo:    connRepo,
	}
}

// Transaction executes a function within a database transaction
func (ds *DatabaseService) Transaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := ds.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()
	
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("tx failed: %v, unable to rollback: %v", err, rbErr)
		}
		return err
	}
	
	return tx.Commit()
}

// GetDB returns the underlying database connection
// This is temporary until all operations are moved to this service
func (ds *DatabaseService) GetDB() *sqlx.DB {
	return ds.db
}

// Sessions returns the session repository
func (ds *DatabaseService) Sessions() repository.SessionRepository {
	return ds.sessionRepo
}

// Messages returns the message repository
func (ds *DatabaseService) Messages() repository.MessageRepository {
	return ds.messageRepo
}

// Configs returns the config repository
func (ds *DatabaseService) Configs() repository.ConfigRepository {
	return ds.configRepo
}

// Connections returns the connection repository
func (ds *DatabaseService) Connections() repository.ConnectionRepository {
	return ds.connRepo
}