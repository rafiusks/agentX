package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Parse command line flags
	var (
		email    = flag.String("email", "test@example.com", "User email")
		password = flag.String("password", "password123", "User password")
		username = flag.String("username", "testuser", "Username")
		fullName = flag.String("name", "Test User", "Full name")
		role     = flag.String("role", "user", "User role (user, admin, premium)")
	)
	flag.Parse()

	// Connect to database
	db, err := sqlx.Connect("postgres", "host=localhost port=5432 user=agentx password=agentx dbname=agentx sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Generate password hash
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), 12)
	if err != nil {
		log.Fatal("Failed to generate password hash:", err)
	}

	// Create user
	userID := uuid.New()
	ctx := context.Background()
	
	query := `
		INSERT INTO users (id, email, username, full_name, password_hash, email_verified, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (email) DO UPDATE SET
			password_hash = EXCLUDED.password_hash,
			username = EXCLUDED.username,
			full_name = EXCLUDED.full_name,
			role = EXCLUDED.role,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	var resultID uuid.UUID
	err = db.GetContext(ctx, &resultID, query,
		userID, *email, *username, *fullName, string(hash), true, *role, time.Now(), time.Now())
	
	if err != nil {
		log.Fatal("Failed to create user:", err)
	}

	if resultID == userID {
		fmt.Printf("✅ Successfully created user:\n")
	} else {
		fmt.Printf("✅ Successfully updated existing user:\n")
	}
	
	fmt.Printf("   Email: %s\n", *email)
	fmt.Printf("   Password: %s\n", *password)
	fmt.Printf("   Username: %s\n", *username)
	fmt.Printf("   Role: %s\n", *role)
	fmt.Printf("   ID: %s\n", resultID)
	fmt.Printf("\nYou can now log in with these credentials!\n")
}