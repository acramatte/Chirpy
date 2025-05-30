package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

func HashPassword(password string) (string, error) {
	fromPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return "", err
	}
	return string(fromPassword), nil
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
	})
	return token.SignedString([]byte(tokenSecret))
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.UUID{}, err
	}
	expirationTime, err := token.Claims.GetExpirationTime()
	if err != nil {
		return uuid.UUID{}, err
	}
	if expirationTime.Time.Before(time.Now().UTC()) {
		return uuid.UUID{}, errors.New("token has expired")
	}
	userID, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, err
	}
	parsedUUID, err := uuid.Parse(userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	return parsedUUID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("No Authorization header found")
	}
	tokenString, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found {
		return "", errors.New("No Authorization header found")
	}
	return tokenString, nil
}

// MakeRefreshToken generates a random 256-bit (32-byte) hex-encoded string.
func MakeRefreshToken() (string, error) {
	// Create a byte slice of length 32
	token := make([]byte, 32)

	// Read random data into the byte slice
	_, err := rand.Read(token)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(token), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("No Authorization header found")
	}
	tokenString, found := strings.CutPrefix(authHeader, "ApiKey ")
	if !found {
		return "", errors.New("No Authorization ApiKey header found")
	}
	return tokenString, nil
}
