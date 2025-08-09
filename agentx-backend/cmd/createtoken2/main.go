package main

import (
	"database/sql"
	"fmt"
	"time"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to database to create a session
	db, err := sql.Open("postgres", "host=localhost port=5432 user=agentx password=agentx dbname=agentx sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create an auth session for the user
	userID := "2162ef6e-c00a-4d75-9f6c-80d2dc759a07"
	sessionID := uuid.New()
	
	_, err = db.Exec(`
		INSERT INTO auth_sessions (id, user_id, access_token_hash, refresh_token_hash, expires_at, created_at, updated_at)
		VALUES ($1, $2, 'dummy', 'dummy', $3, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET updated_at = NOW()
	`, sessionID, userID, time.Now().Add(24*time.Hour))
	if err != nil {
		fmt.Printf("Warning: Could not create session: %v\n", err)
	}

	// Use the same secret as the backend
	secret := []byte("change-me-in-production")
	
	// Create token with all required fields
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":    userID,
		"email":      "vidal@live.com", 
		"username":   "vidal",
		"role":       "user",
		"session_id": sessionID.String(),
		"token_type": "access",
		"exp":        time.Now().Add(24 * time.Hour).Unix(),
		"iat":        time.Now().Unix(),
		"iss":        "agentx",
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		panic(err)
	}

	fmt.Println("Access token for vidal@live.com:")
	fmt.Println(tokenString)
	fmt.Println("\nSession ID:", sessionID.String())
	fmt.Println("\nAdd this to localStorage in the browser console:")
	fmt.Printf("localStorage.setItem('access_token', '%s');\n", tokenString)
}