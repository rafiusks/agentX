package main

import (
	"fmt"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Use the same secret as the backend
	secret := []byte("change-me-in-production")
	
	// Create token for vidal@live.com
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "2162ef6e-c00a-4d75-9f6c-80d2dc759a07",
		"email":   "vidal@live.com",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})

	tokenString, err := token.SignedString(secret)
	if err != nil {
		panic(err)
	}

	fmt.Println("Access token for vidal@live.com:")
	fmt.Println(tokenString)
	fmt.Println("\nAdd this to localStorage in the browser console:")
	fmt.Printf("localStorage.setItem('access_token', '%s');\n", tokenString)
}