package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	// "strings" // Removed unused import
	"testing"
	"time" 

	"github.com/acramatte/Chirpy/internal/auth" 
	"github.com/acramatte/Chirpy/internal/database"
	"github.com/google/uuid"
)

// NOTE: MockAuthFunc types (mockValidateJWTFunc, etc.), MockDB, 
// TestHelperApiConfig, and createActualApiConfig are assumed to be defined 
// in another _test.go file in package main (e.g., handler_chirps_test.go)
// and are thus available here. If not, they would need to be defined or imported.


// --- Global Test Data specific to login tests ---
var testLoginUser = database.User{ 
	ID:          uuid.New(),
	Email:       "logintest@example.com",
	IsChirpyRed: false,
	// HashedPassword will be set using auth.HashPassword in test setup
}
var testLoginPassword = "password123"


// --- Test Functions ---

func TestHandlerLogin(t *testing.T) {
	hashedTestPassword, err := auth.HashPassword(testLoginPassword)
	if err != nil {
		t.Fatalf("Failed to hash test password: %v", err)
	}
	userForLoginTest := testLoginUser 
	userForLoginTest.HashedPassword = hashedTestPassword

	type requestBody struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type responseData struct { 
		ID           uuid.UUID `json:"id"`
		Email        string    `json:"email"`
		IsChirpyRed  bool      `json:"is_chirpy_red"`
		Token        string    `json:"token"`        
		RefreshToken string    `json:"refresh_token"` 
	}

	tests := []struct {
		name                string
		reqBody             requestBody
		setupMockDB         func(*MockDB) 
		expectedStatusCode  int
		expectTokenFields   bool 
	}{
		{
			name: "Success Case",
			reqBody: requestBody{Email: userForLoginTest.Email, Password: testLoginPassword},
			setupMockDB: func(mdb *MockDB) {
				mdb.GetUserByEmailFunc = func(ctx context.Context, email string) (database.User, error) {
					if email == userForLoginTest.Email { return userForLoginTest, nil }
					return database.User{}, sql.ErrNoRows
				}
				mdb.CreateRefreshTokenFunc = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					if arg.UserID != userForLoginTest.ID { t.Errorf("CreateRefreshToken UserID mismatch: got %v, want %v", arg.UserID, userForLoginTest.ID) }
					if arg.Token == "" { t.Error("CreateRefreshToken received empty token string") }
					return database.RefreshToken{Token: arg.Token, UserID: arg.UserID, ExpiresAt: arg.ExpiresAt, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
				}
			},
			expectedStatusCode: http.StatusOK,
			expectTokenFields:  true,
		},
		{
			name: "User Not Found",
			reqBody: requestBody{Email: "notfound@example.com", Password: "password"},
			setupMockDB: func(mdb *MockDB) {
				mdb.GetUserByEmailFunc = func(ctx context.Context, email string) (database.User, error) {
					return database.User{}, sql.ErrNoRows
				}
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name: "Incorrect Password",
			reqBody: requestBody{Email: userForLoginTest.Email, Password: "wrongpassword"},
			setupMockDB: func(mdb *MockDB) {
				mdb.GetUserByEmailFunc = func(ctx context.Context, email string) (database.User, error) {
					if email == userForLoginTest.Email { return userForLoginTest, nil } 
					return database.User{}, sql.ErrNoRows
				}
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name: "DB Error on CreateRefreshToken",
			reqBody: requestBody{Email: userForLoginTest.Email, Password: testLoginPassword},
			setupMockDB: func(mdb *MockDB) {
				mdb.GetUserByEmailFunc = func(ctx context.Context, email string) (database.User, error) { return userForLoginTest, nil }
				mdb.CreateRefreshTokenFunc = func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) {
					return database.RefreshToken{}, errors.New("DB error creating refresh token")
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name: "Malformed JSON Input",
			reqBody: requestBody{}, 
			expectedStatusCode: http.StatusInternalServerError, 
		},
		// Note: Testing MakeJWT/MakeRefreshToken direct failures is hard.
		// If MakeJWT fails, handler returns 500. If MakeRefreshToken fails, handler returns 500.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			helperCfg := newTestHelperApiConfig(mockDB) // Assumes this is defined in handler_chirps_test.go
			cfgToPass := createActualApiConfig(helperCfg) // Assumes this is defined in handler_chirps_test.go

			var reqBodyReader io.Reader
			if tt.name == "Malformed JSON Input" {
				reqBodyReader = bytes.NewBufferString("not-json")
			} else {
				jsonBody, _ := json.Marshal(tt.reqBody)
				reqBodyReader = bytes.NewBuffer(jsonBody)
			}
			
			req := httptest.NewRequest(http.MethodPost, "/api/login", reqBodyReader)
			rr := httptest.NewRecorder()
			cfgToPass.handlerLogin(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("Status: got %d, want %d. Body: %s", rr.Code, tt.expectedStatusCode, rr.Body.String())
			}

			if tt.expectTokenFields && rr.Code == http.StatusOK {
				var resp responseData
				if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil { t.Fatalf("Unmarshal error: %v. Body: %s", err, rr.Body.String()) }
				if resp.ID != userForLoginTest.ID { t.Errorf("ID: got %v, want %v", resp.ID, userForLoginTest.ID) }
				if resp.Email != userForLoginTest.Email { t.Errorf("Email: got %s, want %s", resp.Email, userForLoginTest.Email) }
				if resp.IsChirpyRed != userForLoginTest.IsChirpyRed { t.Errorf("IsChirpyRed: got %v, want %v", resp.IsChirpyRed, userForLoginTest.IsChirpyRed) }
				if resp.Token == "" { t.Error("Expected non-empty JWT token") }
				if resp.RefreshToken == "" { t.Error("Expected non-empty refresh token") }
			}
		})
	}
}

func TestHandlerRefresh(t *testing.T) {
	type response struct { Token string `json:"token"` }
	validRefreshToken := "testrefreshtoken"
	userForRefreshTest := testLoginUser 

	tests := []struct {
		name string
		tokenToSend string 
		mockGetBearerToken mockGetBearerTokenFunc 
		setupMockDB func(*MockDB)
		expectedStatusCode int
		expectTokenInResponse bool
	}{
		{
			name: "Success Case",
			tokenToSend: validRefreshToken,
			mockGetBearerToken: func(headers http.Header) (string, error) { return validRefreshToken, nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.GetUserFromRefreshTokenFunc = func(ctx context.Context, token string) (database.User, error) {
					if token == validRefreshToken { return userForRefreshTest, nil } 
					return database.User{}, errors.New("invalid refresh token in DB mock")
				}
			},
			expectedStatusCode: http.StatusOK,
			expectTokenInResponse: true,
		},
		{
			name: "No Token Found (GetBearerToken error)",
			tokenToSend: "", 
			mockGetBearerToken: func(headers http.Header) (string, error) { return "", errors.New("auth: no token found by mock") },
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "Invalid/Expired Refresh Token (DB error)",
			tokenToSend: "expiredOrInvalidToken",
			mockGetBearerToken: func(headers http.Header) (string, error) { return "expiredOrInvalidToken", nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.GetUserFromRefreshTokenFunc = func(ctx context.Context, token string) (database.User, error) {
					return database.User{}, errors.New("DB: token not found or expired")
				}
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		// Note: Forcing auth.MakeJWT to fail here is difficult. Handler returns 401 if MakeJWT fails.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			helperCfg := newTestHelperApiConfig(mockDB) 
			if tt.mockGetBearerToken != nil { helperCfg.GetBearerTokenFunc = tt.mockGetBearerToken }
			cfgToPass := createActualApiConfig(helperCfg) 

			req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
			if tt.tokenToSend != "" {
				req.Header.Set("Authorization", "Bearer "+tt.tokenToSend)
			}
			
			rr := httptest.NewRecorder()
			cfgToPass.handlerRefresh(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("Status: got %d, want %d. Body: %s", rr.Code, tt.expectedStatusCode, rr.Body.String())
			}
			if tt.expectTokenInResponse && rr.Code == http.StatusOK {
				var respBody response
				if err := json.Unmarshal(rr.Body.Bytes(), &respBody); err != nil { t.Fatalf("Unmarshal error: %v", err) }
				if respBody.Token == "" { t.Error("Expected new JWT token in response, got empty") }
			}
		})
	}
}

func TestHandlerRevoke(t *testing.T) {
	tokenToRevoke := "testrevoketoken"

	tests := []struct {
		name string
		tokenToSend string
		mockGetBearerToken mockGetBearerTokenFunc
		setupMockDB func(*MockDB)
		expectedStatusCode int
	}{
		{
			name: "Success Case",
			tokenToSend: tokenToRevoke,
			mockGetBearerToken: func(headers http.Header) (string, error) { return tokenToRevoke, nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.RevokeTokenFunc = func(ctx context.Context, token string) error {
					if token == tokenToRevoke { return nil }
					return errors.New("unexpected token to revoke")
				}
			},
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name: "No Token Found (GetBearerToken error)",
			tokenToSend: "",
			mockGetBearerToken: func(headers http.Header) (string, error) { return "", errors.New("auth: no token found by mock") },
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "Database Error on Revoke",
			tokenToSend: tokenToRevoke,
			mockGetBearerToken: func(headers http.Header) (string, error) { return tokenToRevoke, nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.RevokeTokenFunc = func(ctx context.Context, token string) error {
					return errors.New("DB error revoking token")
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			helperCfg := newTestHelperApiConfig(mockDB) 
			if tt.mockGetBearerToken != nil { helperCfg.GetBearerTokenFunc = tt.mockGetBearerToken }
			cfgToPass := createActualApiConfig(helperCfg)

			req := httptest.NewRequest(http.MethodPost, "/api/revoke", nil)
			if tt.tokenToSend != "" {
				req.Header.Set("Authorization", "Bearer "+tt.tokenToSend)
			}
			
			rr := httptest.NewRecorder()
			cfgToPass.handlerRevoke(rr, req)

			if rr.Code != tt.expectedStatusCode {
				t.Errorf("Status: got %d, want %d. Body: %s", rr.Code, tt.expectedStatusCode, rr.Body.String())
			}
		})
	}
}
