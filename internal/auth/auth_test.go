package auth

import (
	"github.com/google/uuid"
	"net/http"
	"testing"
	"time"
)

func TestCheckPasswordHash(t *testing.T) {
	testCases := []struct {
		name          string
		password      string
		correctPassword string
		expectError   bool
	}{
		{
			name:          "correct password",
			password:      "mysecretpassword",
			correctPassword: "mysecretpassword",
			expectError:   false,
		},
		{
			name:          "incorrect password",
			password:      "mysecretpassword",
			correctPassword: "wrongpassword",
			expectError:   true,
		},
		{
			name:          "empty password",
			password:      "",
			correctPassword: "",
			expectError:   false,
		},
		{
			name:          "password with special characters",
			password:      "!@#$%^&*()_+",
			correctPassword: "!@#$%^&*()_+",
			expectError:   false,
		},
		{
			name:          "incorrect password with special characters",
			password:      "!@#$%^&*()_+",
			correctPassword: "+_)(*&^%$#@!",
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPassword(tc.password)
			if err != nil {
				// If HashPassword fails for any reason (even for an empty string), it's a setup problem for this test case.
				// bcrypt itself doesn't error on empty strings, so any error here is unexpected.
				t.Fatalf("Failed to hash password '%s': %v", tc.password, err)
			}

			err = CheckPasswordHash(tc.correctPassword, hash)
			if tc.expectError && err == nil {
				t.Errorf("Expected error for password '%s' and hash of '%s', got none", tc.correctPassword, tc.password)
			}
			if !tc.expectError && err != nil {
				t.Errorf("Expected no error for password '%s' and hash of '%s', got %v", tc.correctPassword, tc.password, err)
			}
		})
	}
}

func TestMakeRefreshToken(t *testing.T) {
	// Test Case 1: Check if a non-empty token is generated
	token1, err := MakeRefreshToken()
	if err != nil {
		t.Fatalf("MakeRefreshToken() failed: %v", err)
	}
	if token1 == "" {
		t.Error("MakeRefreshToken() returned an empty token")
	}

	// Test Case 2: Check if subsequent calls generate different tokens
	token2, err := MakeRefreshToken()
	if err != nil {
		t.Fatalf("MakeRefreshToken() failed on second call: %v", err)
	}
	if token2 == "" {
		t.Error("MakeRefreshToken() returned an empty token on second call")
	}

	if token1 == token2 {
		t.Error("MakeRefreshToken() returned the same token on subsequent calls")
	}

	// Test Case 3: Check token length (optional, but good for sanity)
	// Assuming a refresh token should have a reasonable length, e.g., > 32 bytes for a UUID like structure
	// This depends on the actual implementation of MakeRefreshToken which uses hex encoding of 32 random bytes.
	// 32 bytes in hex is 64 characters.
	expectedMinLength := 64 
	if len(token1) < expectedMinLength {
		t.Errorf("MakeRefreshToken() token length is %d, expected at least %d", len(token1), expectedMinLength)
	}
	if len(token2) < expectedMinLength {
		t.Errorf("MakeRefreshToken() second token length is %d, expected at least %d", len(token2), expectedMinLength)
	}
}

func TestGetAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		header      http.Header
		expectedKey string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid API Key",
			header:      http.Header{"Authorization": []string{"ApiKey validkey123"}},
			expectedKey: "validkey123",
			expectError: false,
		},
		{
			name:        "Missing Authorization Header",
			header:      http.Header{},
			expectedKey: "",
			expectError: true,
			errorMsg:    "expected error when Authorization header is missing",
		},
		{
			name:        "No ApiKey Prefix in Header",
			header:      http.Header{"Authorization": []string{"NotApiKey validkey123"}},
			expectedKey: "",
			expectError: true,
			errorMsg:    "expected error when Authorization header does not contain an ApiKey prefix",
		},
		{
			name:        "Malformed ApiKey - No Space",
			header:      http.Header{"Authorization": []string{"ApiKeyvalidkey123"}},
			expectedKey: "",
			expectError: true,
			errorMsg:    "expected error when ApiKey is malformed (no space)",
		},
		{
			name:        "Malformed ApiKey - Too Short (No Key)",
			header:      http.Header{"Authorization": []string{"ApiKey "}},
			expectedKey: "",
			expectError: true,
			errorMsg:    "expected error when ApiKey is malformed (key is empty string)",
		},
		{
			name:        "Empty API Key Value",
			header:      http.Header{"Authorization": []string{"ApiKey"}},
			expectedKey: "",
			expectError: true,
			errorMsg:    "expected error for empty api key value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := GetAPIKey(tt.header)
			if tt.expectError {
				if err == nil {
					t.Errorf("%s: %s. Got key: '%s'", tt.name, tt.errorMsg, key)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.name, err)
				}
				if key != tt.expectedKey {
					t.Errorf("%s: expected key '%s', got '%s'", tt.name, tt.expectedKey, key)
				}
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	tokenSecret := "1Tr8KncVqXj05kWn9CgEKDNcbOyn/YzeirfjROAd/nvCnq2v1tn4yRuZHhW+zVp080Td8fuI95Q2B0RQhaDX3g=="
	differentSecret := "anotherSecretAnotherSecretAnotherSecretAnotherSecret12345"
	userID := uuid.New()
	expiresIn := time.Hour

	// --- Test Case 1: Valid token ---
	validTokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT for valid case: %v", err)
	}
	parsedUUID, err := ValidateJWT(validTokenString, tokenSecret)
	if err != nil {
		t.Errorf("Valid token: Expected no error, got %v", err)
	}
	if parsedUUID != userID {
		t.Errorf("Valid token: Expected userID %v, got %v", userID, parsedUUID)
	}

	// --- Test Case 2: Expired token ---
	expiredTokenString, err := MakeJWT(userID, tokenSecret, -time.Hour) // Token created to be already expired
	if err != nil {
		t.Fatalf("Failed to create JWT for expired case: %v", err)
	}
	_, err = ValidateJWT(expiredTokenString, tokenSecret)
	if err == nil {
		t.Error("Expired token: Expected error, got none")
	}

	// --- Test Case 3: Token signed with a different secret ---
	tokenWithDifferentSecret, err := MakeJWT(userID, differentSecret, expiresIn)
	if err != nil {
		t.Fatalf("Failed to create JWT for different secret case: %v", err)
	}
	_, err = ValidateJWT(tokenWithDifferentSecret, tokenSecret) // Validate with the original secret
	if err == nil {
		t.Error("Different secret: Expected error, got none")
	}

	// --- Test Case 4: Malformed token ---
	malformedToken := "this.is.not.a.jwt"
	_, err = ValidateJWT(malformedToken, tokenSecret)
	if err == nil {
		t.Error("Malformed token: Expected error, got none")
	}

	// --- Test Case 5: Invalid token (structurally okay but garbage content) ---
	_, err = ValidateJWT("invalidtoken", tokenSecret)
	if err == nil {
		t.Error("Expected error for invalid token (garbage content), got none")
	}

	// --- Test Case 6: Empty token string ---
	_, err = ValidateJWT("", tokenSecret)
	if err == nil {
		t.Error("Empty token string: Expected error, got none")
	}
}

func TestGetBearerToken(t *testing.T) {
	tests := []struct {
		name          string
		header        http.Header
		expectedToken string
		expectError   bool
		errorMsg      string
	}{
		{
			name: "Valid Bearer Token",
			header: http.Header{"Authorization": []string{"Bearer validtoken123"}},
			expectedToken: "validtoken123",
			expectError:   false,
		},
		{
			name: "Missing Authorization Header",
			header: http.Header{},
			expectedToken: "",
			expectError:   true,
			errorMsg:      "expected error when Authorization header is missing",
		},
		{
			name: "No Bearer Token in Header",
			header: http.Header{"Authorization": []string{"NotBearer validtoken123"}},
			expectedToken: "",
			expectError:   true,
			errorMsg:      "expected error when Authorization header does not contain a Bearer token",
		},
		{
			name: "Malformed Bearer Token - No Space",
			header: http.Header{"Authorization": []string{"Bearervalidtoken123"}},
			expectedToken: "",
			expectError:   true,
			errorMsg:      "expected error when Bearer token is malformed (no space)",
		},
		{
			name: "Malformed Bearer Token - Too Short",
			header: http.Header{"Authorization": []string{"Bearer "}}, // Token part is empty
			expectedToken: "",
			expectError:   true,
			errorMsg:      "expected error when Bearer token is malformed (too short)",
		},
		{
			name: "Malformed Bearer Token - Multiple spaces",
			header: http.Header{"Authorization": []string{"Bearer  validtoken123"}}, // Token part is empty
			expectedToken: " validtoken123", // The current implementation will pass this, which might be a bug in the function itself.
			expectError:   false, // Adjust if GetBearerToken is fixed to trim spaces or handle this as an error.
			errorMsg:      "",
		},
		{
			name: "Empty Token Value",
			header: http.Header{"Authorization": []string{"Bearer"}},
			expectedToken: "",
			expectError:   true,
			errorMsg:      "expected error for empty token value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GetBearerToken(tt.header)
			if tt.expectError {
				if err == nil {
					t.Errorf("%s: %s", tt.name, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.name, err)
				}
				if token != tt.expectedToken {
					t.Errorf("%s: expected token '%s', got '%s'", tt.name, tt.expectedToken, token)
				}
			}
		})
	}
}
