package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	subject := flag.String(
		"subject",
		"development-user",
		"JWT subject identifying the user",
	)

	ttl := flag.Duration(
		"ttl",
		time.Hour,
		"token lifetime",
	)

	flag.Parse()

	if err := run(*subject, *ttl); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(subject string, ttl time.Duration) error {
	secret := os.Getenv("JWT_SECRET")

	if len(secret) < 32 {
		return fmt.Errorf(
			"JWT_SECRET must contain at least 32 characters",
		)
	}

	if subject == "" {
		return fmt.Errorf("subject must not be empty")
	}

	if ttl <= 0 {
		return fmt.Errorf("ttl must be greater than zero")
	}

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	signedToken, err := token.SignedString(
		[]byte(secret),
	)
	if err != nil {
		return fmt.Errorf("sign JWT: %w", err)
	}

	fmt.Println(signedToken)

	return nil
}
