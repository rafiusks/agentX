package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Connect to database
	db, err := sql.Open("postgres", "host=localhost port=5432 user=agentx password=agentx dbname=agentx sslmode=disable")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// User details
	userID := uuid.New()
	email := "test@test.test"
	username := "testuser"
	password := "test123"

	// Generate password hash
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal("Failed to generate hash:", err)
	}

	// Check if user exists
	var existingID string
	err = db.QueryRow("SELECT id FROM users WHERE email = $1", email).Scan(&existingID)
	
	if err == sql.ErrNoRows {
		// User doesn't exist, create it
		err = db.QueryRow(`
			INSERT INTO users (id, email, username, password_hash, full_name, email_verified, is_active, role, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			RETURNING id
		`, userID, email, username, string(hash), "Test User", true, true, "user").Scan(&existingID)
		
		if err != nil {
			log.Fatal("Failed to create user:", err)
		}
		fmt.Printf("Created user with ID: %s\n", existingID)
	} else if err != nil {
		log.Fatal("Database error:", err)
	} else {
		// User exists, update password
		userID = uuid.MustParse(existingID)
		_, err = db.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hash), userID)
		if err != nil {
			log.Fatal("Failed to update password:", err)
		}
		fmt.Printf("Updated existing user with ID: %s\n", userID)
	}

	// Create JWT token
	secret := []byte("change-me-in-production")
	sessionID := uuid.New()
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    userID.String(),
		"email":      email,
		"username":   username,
		"role":       "user",
		"session_id": sessionID.String(),
		"token_type": "access",
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
		"iss":        "agentx",
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		log.Fatal("Failed to generate token:", err)
	}

	fmt.Println("\n=== Test User Created Successfully ===")
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("User ID: %s\n", userID)
	fmt.Printf("Session ID: %s\n", sessionID)
	fmt.Println("\n=== JWT Token ===")
	fmt.Println(tokenString)
	fmt.Println("\n=== Use in Browser Console ===")
	fmt.Printf("localStorage.setItem('access_token', '%s');\n", tokenString)
}