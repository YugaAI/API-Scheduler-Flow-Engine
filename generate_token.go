package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	secret := os.Getenv("JWT_SECRET")
	secretByte := []byte(secret)
	claims := jwt.MapClaims{
		"sub":  "123e4567-e89b-12d3-a456-426614174000",
		"role": "ADMIN",
		"exp":  time.Now().Add(time.Hour * 24 * 365).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(secretByte)
	fmt.Print(tokenString)
}
