package main

import (
	"database/sql"
	"fmt"
	"log"

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

	// Generate password hash
	password := "password123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal("Failed to generate hash:", err)
	}

	fmt.Printf("Generated hash for '%s': %s\n", password, string(hash))

	// Update user password
	email := "vidal@live.com"
	_, err = db.Exec("UPDATE users SET password_hash = $1, updated_at = NOW() WHERE email = $2", string(hash), email)
	if err != nil {
		log.Fatal("Failed to update password:", err)
	}

	fmt.Printf("Successfully updated password for %s\n", email)
}