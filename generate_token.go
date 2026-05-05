package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := []byte("supersecretjwtkeythatshouldbechanged")
	claims := jwt.MapClaims{
		"sub":  "123e4567-e89b-12d3-a456-426614174000",
		"role": "ADMIN",
		"exp":  time.Now().Add(time.Hour * 24 * 365).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secret)
	fmt.Print(tokenString)
}
