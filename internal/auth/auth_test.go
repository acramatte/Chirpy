package auth

import (
	"github.com/google/uuid"
	"net/http"
	"testing"
	"time"
)

func TestCheckPasswordHash(t *testing.T) {
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test with the correct password
	err = CheckPasswordHash(password, hash)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test with an incorrect password
	err = CheckPasswordHash("wrongpassword", hash)
	if err == nil {
		t.Error("Expected error for incorrect password, got none")
	}
}

func TestValidateJWT(t *testing.T) {
	tokenSecret := "1Tr8KncVqXj05kWn9CgEKDNcbOyn/YzeirfjROAd/nvCnq2v1tn4yRuZHhW+zVp080Td8fuI95Q2B0RQhaDX3g=="
	userID := uuid.New()
	expiresIn := time.Hour

	// Create a valid JWT token
	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Test with a valid token
	parsedUUID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if parsedUUID != userID {
		t.Errorf("Expected userID %v, got %v", userID, parsedUUID)
	}

	// Test with an invalid token
	_, err = ValidateJWT("invalidtoken", tokenSecret)
	if err == nil {
		t.Error("Expected error for invalid token, got none")
	}

	// Test with an expired token
	expiredTokenString, err := MakeJWT(userID, tokenSecret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to create expired JWT: %v", err)
	}
	_, err = ValidateJWT(expiredTokenString, tokenSecret)
	if err == nil {
		t.Error("Expected error for expired token, got none")
	}
}

func TestGetBearerToken(t *testing.T) {
	header := http.Header{}
	header.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
	token, err := GetBearerToken(header)
	if err != nil {
		return
	}
	if token != "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9" {
		t.Error("TestGetBearerToken() error - token parsed not matching")
	}
}
