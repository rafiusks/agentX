package db

import (
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jmoiron/sqlx"
)

// Database wraps the database connection pool
type Database struct {
	Pool interface{} // Can be *pgxpool.Pool or *sqlx.DB
}

// NewDatabase creates a new Database wrapper
func NewDatabase(pool interface{}) *Database {
	return &Database{Pool: pool}
}

// NewDatabaseFromSQLX creates a new Database wrapper from sqlx.DB
func NewDatabaseFromSQLX(db *sqlx.DB) *Database {
	return &Database{Pool: db}
}

// NewDatabaseFromPGX creates a new Database wrapper from pgxpool.Pool
func NewDatabaseFromPGX(pool *pgxpool.Pool) *Database {
	return &Database{Pool: pool}
}