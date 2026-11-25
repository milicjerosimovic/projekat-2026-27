package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" || len(os.Args) < 2 {
		fmt.Println("Upotreba: set JWT_SECRET=... pa: go run ./cmd/mktoken <userId>")
		os.Exit(1)
	}
	now := time.Now().UTC()
	c := claims{
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   os.Args[1],
			ID:        "probni-token",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(secret))
	if err != nil {
		fmt.Println("Greška:", err)
		os.Exit(1)
	}
	fmt.Println(token)
}
