package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort" 
	"testing"
	"time"

	"github.com/acramatte/Chirpy/internal/database"
	"github.com/go-chi/chi/v5" 
	"github.com/google/uuid"
)

// MockAuthFunc types
type mockValidateJWTFunc func(tokenString, tokenSecret string) (uuid.UUID, error)
type mockGetBearerTokenFunc func(headers http.Header) (string, error)
type mockGetAPIKeyFunc func(headers http.Header) (string, error)

// MockDB implements database.Querier
type MockDB struct {
	CreateChirpFunc             func(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error)
	DeleteChirpFunc             func(ctx context.Context, id uuid.UUID) error
	GetChirpFunc                func(ctx context.Context, id uuid.UUID) (database.Chirp, error)
	GetChirpsFunc               func(ctx context.Context, dollar_1 interface{}) ([]database.Chirp, error)
	GetChirpsByAuthorIdFunc     func(ctx context.Context, arg database.GetChirpsByAuthorIdParams) ([]database.Chirp, error)
	CreateUserFunc              func(ctx context.Context, arg database.CreateUserParams) (database.User, error)
	DeleteAllFunc               func(ctx context.Context) error
	GetUserByEmailFunc          func(ctx context.Context, email string) (database.User, error)
	UpdateEmailAndPasswordFunc  func(ctx context.Context, arg database.UpdateEmailAndPasswordParams) (database.User, error)
	UpgradeToRedFunc            func(ctx context.Context, id uuid.UUID) (database.User, error)
	CreateRefreshTokenFunc      func(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error)
	GetRefreshTokenFunc         func(ctx context.Context, token string) (database.RefreshToken, error)
	GetUserFromRefreshTokenFunc func(ctx context.Context, token string) (database.User, error)
	RevokeTokenFunc             func(ctx context.Context, token string) error
}

// Methods for database.Querier
func (m *MockDB) CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) { if m.CreateChirpFunc != nil { return m.CreateChirpFunc(ctx, arg) }; return database.Chirp{}, errors.New("MockDB: CreateChirpFunc not set") }
func (m *MockDB) DeleteChirp(ctx context.Context, id uuid.UUID) error { if m.DeleteChirpFunc != nil { return m.DeleteChirpFunc(ctx, id) }; return errors.New("MockDB: DeleteChirpFunc not set") }
func (m *MockDB) GetChirp(ctx context.Context, id uuid.UUID) (database.Chirp, error) { if m.GetChirpFunc != nil { return m.GetChirpFunc(ctx, id) }; return database.Chirp{}, errors.New("MockDB: GetChirpFunc not set") }
func (m *MockDB) GetChirps(ctx context.Context, dollar_1 interface{}) ([]database.Chirp, error) { if m.GetChirpsFunc != nil { return m.GetChirpsFunc(ctx, dollar_1) }; return nil, errors.New("MockDB: GetChirpsFunc not set") }
func (m *MockDB) GetChirpsByAuthorId(ctx context.Context, arg database.GetChirpsByAuthorIdParams) ([]database.Chirp, error) { if m.GetChirpsByAuthorIdFunc != nil { return m.GetChirpsByAuthorIdFunc(ctx, arg) }; return nil, errors.New("MockDB: GetChirpsByAuthorIdFunc not set") }
func (m *MockDB) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) { if m.CreateUserFunc != nil {return m.CreateUserFunc(ctx,arg)}; return database.User{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) DeleteAll(ctx context.Context) error { if m.DeleteAllFunc != nil {return m.DeleteAllFunc(ctx)}; return errors.New("not implemented by chirp mock") }
func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (database.User, error) { if m.GetUserByEmailFunc != nil {return m.GetUserByEmailFunc(ctx,email)}; return database.User{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) UpdateEmailAndPassword(ctx context.Context, arg database.UpdateEmailAndPasswordParams) (database.User, error) { if m.UpdateEmailAndPasswordFunc != nil {return m.UpdateEmailAndPasswordFunc(ctx,arg)}; return database.User{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) UpgradeToRed(ctx context.Context, id uuid.UUID) (database.User, error) { if m.UpgradeToRedFunc != nil {return m.UpgradeToRedFunc(ctx,id)}; return database.User{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) CreateRefreshToken(ctx context.Context, arg database.CreateRefreshTokenParams) (database.RefreshToken, error) { if m.CreateRefreshTokenFunc != nil {return m.CreateRefreshTokenFunc(ctx,arg)}; return database.RefreshToken{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) GetRefreshToken(ctx context.Context, token string) (database.RefreshToken, error) { if m.GetRefreshTokenFunc != nil {return m.GetRefreshTokenFunc(ctx,token)}; return database.RefreshToken{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) GetUserFromRefreshToken(ctx context.Context, token string) (database.User, error) { if m.GetUserFromRefreshTokenFunc != nil {return m.GetUserFromRefreshTokenFunc(ctx,token)}; return database.User{}, errors.New("not implemented by chirp mock") }
func (m *MockDB) RevokeToken(ctx context.Context, token string) error { if m.RevokeTokenFunc != nil {return m.RevokeTokenFunc(ctx,token)}; return errors.New("not implemented by chirp mock") }
var _ database.Querier = (*MockDB)(nil)

type TestHelperApiConfig struct {
	DB                  database.Querier
	JwtSecret           string
	PolkaWebhookKey     string
	ValidateJWTFunc     mockValidateJWTFunc
	GetBearerTokenFunc  mockGetBearerTokenFunc
	GetAPIKeyFunc       mockGetAPIKeyFunc
}

func newTestHelperApiConfig(db *MockDB) *TestHelperApiConfig {
	return &TestHelperApiConfig{
		DB:              db,
		JwtSecret:       "test-jwt-secret",
		PolkaWebhookKey: "test-polka-key",
		ValidateJWTFunc: func(tokenString, tokenSecret string) (uuid.UUID, error) { return uuid.Nil, errors.New("auth: ValidateJWTFunc not configured") },
		GetBearerTokenFunc: func(headers http.Header) (string, error) { return "", errors.New("auth: GetBearerTokenFunc not configured") },
		GetAPIKeyFunc:      func(headers http.Header) (string, error) { return "", errors.New("auth: GetAPIKeyFunc not configured") },
	}
}

func createActualApiConfig(helper *TestHelperApiConfig) *apiConfig {
	return &apiConfig{
		db:                  helper.DB,
		jwtSecret:           helper.JwtSecret,
		polkaWebhookKey:     helper.PolkaWebhookKey,
		validateJWT:         helper.ValidateJWTFunc,
		getBearerToken:      helper.GetBearerTokenFunc,
		getAPIKey:           helper.GetAPIKeyFunc,
	}
}

var (
	user1ID  = uuid.New()
	user2ID  = uuid.New()
	baseTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	chirp1   = database.Chirp{ID: uuid.New(), UserID: user1ID, Body: "Alpha chirp from User 1", CreatedAt: baseTime.Add(-2 * time.Hour)}
	chirp2   = database.Chirp{ID: uuid.New(), UserID: user2ID, Body: "Beta chirp from User 2", CreatedAt: baseTime.Add(-1 * time.Hour)}
	chirp3   = database.Chirp{ID: uuid.New(), UserID: user1ID, Body: "Gamma chirp from User 1", CreatedAt: baseTime}
	
	allTestChirpsSortedAsc  = []database.Chirp{chirp1, chirp2, chirp3}
	user1ChirpsSortedAsc    = []database.Chirp{chirp1, chirp3}
)

// Helper to get a reversed copy of a chirp slice for testing desc sort
func getReversedChirpsCopy(chirps []database.Chirp) []database.Chirp {
	reversed := make([]database.Chirp, len(chirps))
	copy(reversed, chirps) 
	sort.SliceStable(reversed, func(i, j int) bool {
		return reversed[i].CreatedAt.After(reversed[j].CreatedAt)
	})
	return reversed
}


func TestGetCleanedBody(t *testing.T) {
	tests := []struct{ name, body, expected string }{
		{"clean body", "This is a clean message.", "This is a clean message."},
		{"kerfuffle simple", "kerfuffle", "****"},
		{"kerfuffle in sentence", "This message contains kerfuffle.", "This message contains ****"},
		{"kerfuffle with period final", "This message contains kerfuffle.", "This message contains ****"},
		{"sharbert simple", "sharbert", "****"},
		{"sharbert in sentence", "Sharbert is a bad word.", "**** is a bad word."},
		{"fornax simple", "fornax", "****"},
		{"fornax in sentence with ?", "What about fornax?", "What about ****"},
		{"multiple profane words", "kerfuffle sharbert fornax", "**** **** ****"},
		{"mixed case", "Kerfuffle Sharbert Fornax", "**** **** ****"},
		{"profane word with punctuation attached", "fornax.", "****"}, 
		{"profane word before punctuation", "fornax, indeed", "**** indeed"}, 
		{"profane substring (fornaxation)", "fornaxation is not fornax.", "**** is not ****"},
		{"empty string", "", ""},
		{"profane at start", "sharbert is bad", "**** is bad"},
		{"profane at end", "bad is sharbert", "bad is ****"},
		{"already censored", "**** is bad", "**** is bad"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleaned := getCleanedBody(tt.body)
			if cleaned != tt.expected {
				t.Errorf("getCleanedBody(%q):\ngot  %q\nwant %q", tt.body, cleaned, tt.expected)
			}
		})
	}
}

func TestHandlerChirpsCreate(t *testing.T) {
	type parameters struct{ Body string `json:"body"` }
	
	tests := []struct {
		name                string
		requestBody         interface{}
		mockGetBearerToken  mockGetBearerTokenFunc
		mockValidateJWT     mockValidateJWTFunc 
		setupMockDB         func(*MockDB)
		expectedStatusCode  int
		validateResponse    func(t *testing.T, rr *httptest.ResponseRecorder, originalBody parameters, expectedUserID uuid.UUID)
	}{
		{
			name:        "Success Case",
			requestBody: parameters{Body: "Valid new chirp"},
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT:    func(tokenString, tokenSecret string) (uuid.UUID, error) { return user1ID, nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.CreateChirpFunc = func(ctx context.Context, params database.CreateChirpParams) (database.Chirp, error) {
					if params.UserID != user1ID { t.Fatalf("Mock CreateChirp: UserID mismatch. Got %v, want %v", params.UserID, user1ID) }
					return database.Chirp{ID: uuid.New(), UserID: params.UserID, Body: params.Body, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
				}
			},
			expectedStatusCode: http.StatusCreated,
			validateResponse: func(t *testing.T, rr *httptest.ResponseRecorder, originalBody parameters, expectedUserID uuid.UUID) {
				t.Logf("Raw JSON response for Success Case (Create): %s", rr.Body.String()) 
				var chirpRespJSON map[string]interface{} // Unmarshal into map to check raw user_id
				if err := json.Unmarshal(rr.Body.Bytes(), &chirpRespJSON); err != nil { t.Fatalf("Unmarshal error: %v. Body: %s", err, rr.Body.String()) }
				
				if body, ok := chirpRespJSON["body"].(string); !ok || body != originalBody.Body {
					t.Errorf("body: got %q, want %q", body, originalBody.Body)
				}
				if userIDStr, ok := chirpRespJSON["user_id"].(string); !ok || userIDStr != expectedUserID.String() {
					t.Errorf("user_id: got %q, want %q", userIDStr, expectedUserID.String())
				}
			},
		},
		{
			name:        "Auth Failure - GetBearerToken error",
			requestBody: parameters{Body: "Chirp attempt"},
			mockGetBearerToken: func(headers http.Header) (string, error) { return "", errors.New("auth: no token header") },
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:        "Auth Failure - ValidateJWT error",
			requestBody: parameters{Body: "Chirp attempt"},
			mockGetBearerToken: func(headers http.Header) (string, error) { return "uselesstoken", nil },
			mockValidateJWT:    func(tokenString, tokenSecret string) (uuid.UUID, error) { return uuid.Nil, errors.New("auth: invalid JWT") },
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:        "Input Validation - Chirp too long",
			requestBody: parameters{Body: string(make([]byte, 141))},
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT:    func(tokenString, tokenSecret string) (uuid.UUID, error) { return user1ID, nil }, 
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:        "Database Error on CreateChirp",
			requestBody: parameters{Body: "Good chirp that causes DB error"},
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT:    func(tokenString, tokenSecret string) (uuid.UUID, error) { return user1ID, nil }, 
			setupMockDB: func(mdb *MockDB) {
				mdb.CreateChirpFunc = func(ctx context.Context, params database.CreateChirpParams) (database.Chirp, error) {
					return database.Chirp{}, errors.New("simulated DB error")
				}
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:        "Malformed JSON input",
			requestBody: "this is not valid JSON",
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil }, 
			mockValidateJWT:    func(tokenString, tokenSecret string) (uuid.UUID, error) { return user1ID, nil },
			expectedStatusCode: http.StatusInternalServerError, // Current handler returns 500 for decode errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			helperCfg := newTestHelperApiConfig(mockDB)
			if tt.mockGetBearerToken != nil { helperCfg.GetBearerTokenFunc = tt.mockGetBearerToken }
			if tt.mockValidateJWT != nil { helperCfg.ValidateJWTFunc = tt.mockValidateJWT }
			cfgToPass := createActualApiConfig(helperCfg)

			var bodyBytes []byte
			if str, ok := tt.requestBody.(string); ok { bodyBytes = []byte(str)
			} else { bodyBytes, _ = json.Marshal(tt.requestBody) }
			
			req := httptest.NewRequest(http.MethodPost, "/api/chirps", bytes.NewReader(bodyBytes))
			if tt.mockGetBearerToken != nil {
				token, err := tt.mockGetBearerToken(http.Header{}) 
				if err == nil && token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
			}
			
			rr := httptest.NewRecorder()
			cfgToPass.handlerChirpsCreate(rr, req)

			if rr.Code != tt.expectedStatusCode { t.Errorf("Status: got %d, want %d. Body: %s", rr.Code, tt.expectedStatusCode, rr.Body.String()) }
			if tt.validateResponse != nil {
				originalBodyParams, _ := tt.requestBody.(parameters)
				expectedUserID := uuid.Nil 
				if tt.mockValidateJWT != nil {
					uid, err := tt.mockValidateJWT("dummyToken", cfgToPass.jwtSecret)
					if err == nil { expectedUserID = uid }
				}
				tt.validateResponse(t, rr, originalBodyParams, expectedUserID)
			}
		})
	}
}

func TestHandlerChirpsRetrieve(t *testing.T) {
	tests := []struct {
		name               string
		queryParams        string
		setupMockDB        func(*MockDB)
		expectedStatusCode int
		expectedChirps     []database.Chirp
	}{
		{
			name: "Success - All Chirps, default sort (asc)",
			setupMockDB: func(mdb *MockDB) {
				mdb.GetChirpsFunc = func(ctx context.Context, sortDir interface{}) ([]database.Chirp, error) {
					data := make([]database.Chirp, len(allTestChirpsSortedAsc)); copy(data, allTestChirpsSortedAsc)
					if strSort, ok := sortDir.(string); ok && strSort == "desc" { return getReversedChirpsCopy(data), nil }
					return data, nil
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedChirps:     allTestChirpsSortedAsc,
		},
		{
			name:        "Success - All Chirps, sort=desc",
			queryParams: "sort=desc",
			setupMockDB: func(mdb *MockDB) {
				mdb.GetChirpsFunc = func(ctx context.Context, sortDir interface{}) ([]database.Chirp, error) {
					data := make([]database.Chirp, len(allTestChirpsSortedAsc)); copy(data, allTestChirpsSortedAsc)
					if strSort, ok := sortDir.(string); ok && strSort == "desc" { return getReversedChirpsCopy(data), nil }
					return data, nil
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedChirps:     getReversedChirpsCopy(allTestChirpsSortedAsc),
		},
		{
			name:        "Success - By Author ID (user1ID), default sort (asc)",
			queryParams: "author_id=" + user1ID.String(),
			setupMockDB: func(mdb *MockDB) {
				mdb.GetChirpsByAuthorIdFunc = func(ctx context.Context, params database.GetChirpsByAuthorIdParams) ([]database.Chirp, error) {
					if params.UserID == user1ID { 
						data := make([]database.Chirp, len(user1ChirpsSortedAsc)); copy(data, user1ChirpsSortedAsc)
						if strSort, ok := params.Column2.(string); ok && strSort == "desc" { return getReversedChirpsCopy(data), nil }
						return data, nil
					}
					return nil, errors.New("mock: unexpected author_id")
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedChirps:     user1ChirpsSortedAsc,
		},
		{
			name:        "Success - By Author ID (user1ID), sort=desc",
			queryParams: "author_id=" + user1ID.String() + "&sort=desc",
			setupMockDB: func(mdb *MockDB) {
				mdb.GetChirpsByAuthorIdFunc = func(ctx context.Context, params database.GetChirpsByAuthorIdParams) ([]database.Chirp, error) {
					if params.UserID == user1ID { 
						data := make([]database.Chirp, len(user1ChirpsSortedAsc)); copy(data, user1ChirpsSortedAsc)
						if strSort, ok := params.Column2.(string); ok && strSort == "desc" { return getReversedChirpsCopy(data), nil }
						return data, nil
					}
					return nil, errors.New("mock: unexpected author_id")
				}
			},
			expectedStatusCode: http.StatusOK,
			expectedChirps:     getReversedChirpsCopy(user1ChirpsSortedAsc),
		},
		{
			name: "Database Error - GetChirps",
			setupMockDB: func(mdb *MockDB) { mdb.GetChirpsFunc = func(ctx context.Context, sortDir interface{}) ([]database.Chirp, error) { return nil, errors.New("DB error") }},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:        "Invalid author_id format",
			queryParams: "author_id=not-a-valid-uuid",
			expectedStatusCode: http.StatusInternalServerError, 
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			helperCfg := newTestHelperApiConfig(mockDB)
			cfgToPass := createActualApiConfig(helperCfg)

			reqPath := "/api/chirps"
			if tt.queryParams != "" { reqPath += "?" + tt.queryParams }
			req := httptest.NewRequest(http.MethodGet, reqPath, nil)
			rr := httptest.NewRecorder()
			cfgToPass.handlerChirpsRetrieve(rr, req)

			if rr.Code != tt.expectedStatusCode { t.Fatalf("Status: got %d, want %d. Path: %s. Body: %s", rr.Code, tt.expectedStatusCode, reqPath, rr.Body.String()) }
			if rr.Code == http.StatusOK && tt.expectedChirps != nil {
				var respChirps []database.Chirp
				if err := json.Unmarshal(rr.Body.Bytes(), &respChirps); err != nil { t.Fatalf("Unmarshal err: %v", err) }
				if len(respChirps) != len(tt.expectedChirps) { t.Fatalf("Chirp count: got %d, want %d. Resp: %s", len(respChirps), len(tt.expectedChirps), rr.Body.String()) }
				for i := range tt.expectedChirps {
					if respChirps[i].ID != tt.expectedChirps[i].ID { t.Errorf("Chirp ID at index %d: got %s, want %s", i, respChirps[i].ID, tt.expectedChirps[i].ID) }
				}
			}
		})
	}
}

func TestHandlerChirpGet(t *testing.T) {
	targetChirp := chirp2 
	tests := []struct {
		name                string
		chirpIDParam        string
		setupMockDB         func(*MockDB)
		expectedStatusCode  int
		expectSpecificChirp *database.Chirp
	}{
		{
			name:         "Success Case",
			chirpIDParam: targetChirp.ID.String(),
			setupMockDB:  func(mdb *MockDB) { mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { if id == targetChirp.ID { return targetChirp, nil }; return database.Chirp{}, sql.ErrNoRows }},
			expectedStatusCode:  http.StatusOK,
			expectSpecificChirp: &targetChirp,
		},
		{
			name:               "Invalid Chirp ID Format",
			chirpIDParam:       "not-a-uuid", 
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:         "Chirp Not Found",
			chirpIDParam: uuid.New().String(),
			setupMockDB:  func(mdb *MockDB) { mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { return database.Chirp{}, sql.ErrNoRows }},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:         "Database Error (other than sql.ErrNoRows)",
			chirpIDParam: targetChirp.ID.String(),
			setupMockDB:  func(mdb *MockDB) { mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { return database.Chirp{}, errors.New("DB error") }},
			expectedStatusCode: http.StatusNotFound, // Handler maps all GetChirp errors to 404
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			helperCfg := newTestHelperApiConfig(mockDB)
			cfgToPass := createActualApiConfig(helperCfg)
			
			reqPath := "/api/chirps/" + tt.chirpIDParam
			req := httptest.NewRequest(http.MethodGet, reqPath, nil)
			
			ctx := req.Context()
			rctx := chi.NewRouteContext() // Use chi context for chi.URLParam
			rctx.URLParams.Add("chirpID", tt.chirpIDParam)
			ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)
			
			rr := httptest.NewRecorder()
			cfgToPass.handlerChirpGet(rr, req)

			if rr.Code != tt.expectedStatusCode { t.Fatalf("Status: got %d, want %d. Path: %s, Body: %s", rr.Code, tt.expectedStatusCode, reqPath, rr.Body.String()) }
			if tt.expectSpecificChirp != nil && rr.Code == http.StatusOK {
				var respChirpJSON map[string]interface{}
				if err := json.Unmarshal(rr.Body.Bytes(), &respChirpJSON); err != nil { t.Fatalf("Unmarshal: %v", err) }
				if idStr, _ := respChirpJSON["id"].(string); idStr != tt.expectSpecificChirp.ID.String() {
					t.Errorf("Chirp ID mismatch: got %s, want %s", idStr, tt.expectSpecificChirp.ID.String())
				}
				if bodyStr, _ := respChirpJSON["body"].(string); bodyStr != tt.expectSpecificChirp.Body {
					t.Errorf("Chirp Body mismatch: got %s, want %s", bodyStr, tt.expectSpecificChirp.Body)
				}
			}
		})
	}
}

func TestHandlerChirpDelete(t *testing.T) {
	chirpToDelete := chirp1 
	authorTryingToDelete := user1ID
	nonAuthorTryingToDelete := user2ID

	tests := []struct {
		name                string
		chirpIDParam        string
		mockGetBearerToken  mockGetBearerTokenFunc
		mockValidateJWT     mockValidateJWTFunc
		setupMockDB         func(*MockDB)
		expectedStatusCode  int
	}{
		{
			name:         "Success Case",
			chirpIDParam: chirpToDelete.ID.String(),
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT: func(tokenString, tokenSecret string) (uuid.UUID, error) { return authorTryingToDelete, nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { 
					if id == chirpToDelete.ID { return chirpToDelete, nil }
					return database.Chirp{}, sql.ErrNoRows 
				}
				mdb.DeleteChirpFunc = func(ctx context.Context, id uuid.UUID) error { 
					if id == chirpToDelete.ID { return nil }
					return errors.New("mock DeleteChirp: unexpected ID")
				}
			},
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:         "Auth Failure - GetBearerToken error",
			chirpIDParam: chirpToDelete.ID.String(),
			mockGetBearerToken: func(headers http.Header) (string, error) { return "", errors.New("no token") },
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:         "Auth Failure - ValidateJWT error",
			chirpIDParam: chirpToDelete.ID.String(),
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT: func(tokenString, tokenSecret string) (uuid.UUID, error) { return uuid.Nil, errors.New("invalid JWT") },
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:         "Forbidden - User is not author",
			chirpIDParam: chirpToDelete.ID.String(),
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT: func(tokenString, tokenSecret string) (uuid.UUID, error) { return nonAuthorTryingToDelete, nil },
			setupMockDB:  func(mdb *MockDB) { mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { if id == chirpToDelete.ID { return chirpToDelete, nil }; return database.Chirp{}, sql.ErrNoRows }},
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:         "Chirp Not Found for deletion",
			chirpIDParam: uuid.New().String(), 
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT: func(tokenString, tokenSecret string) (uuid.UUID, error) { return authorTryingToDelete, nil },
			setupMockDB:  func(mdb *MockDB) { mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { return database.Chirp{}, sql.ErrNoRows }},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:         "DB Error on DeleteChirp",
			chirpIDParam: chirpToDelete.ID.String(),
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil },
			mockValidateJWT: func(tokenString, tokenSecret string) (uuid.UUID, error) { return authorTryingToDelete, nil },
			setupMockDB: func(mdb *MockDB) {
				mdb.GetChirpFunc = func(ctx context.Context, id uuid.UUID) (database.Chirp, error) { return chirpToDelete, nil }
				mdb.DeleteChirpFunc = func(ctx context.Context, id uuid.UUID) error { return errors.New("DB delete error") }
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
		{
			name:         "Invalid Chirp ID Format for deletion",
			chirpIDParam: "not-a-uuid",
			mockGetBearerToken: func(headers http.Header) (string, error) { return "validtoken", nil }, 
			mockValidateJWT: func(tokenString, tokenSecret string) (uuid.UUID, error) { return authorTryingToDelete, nil },
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDB{}; if tt.setupMockDB != nil { tt.setupMockDB(mockDB) }
			
			helperCfg := newTestHelperApiConfig(mockDB)
			if tt.mockGetBearerToken != nil { helperCfg.GetBearerTokenFunc = tt.mockGetBearerToken }
			if tt.mockValidateJWT != nil { helperCfg.ValidateJWTFunc = tt.mockValidateJWT }
			cfgToPass := createActualApiConfig(helperCfg)
			
			reqPath := "/api/chirps/" + tt.chirpIDParam
			req := httptest.NewRequest(http.MethodDelete, reqPath, nil)
			
			ctx := req.Context()
			rctx := chi.NewRouteContext() // Use chi context for chi.URLParam
			rctx.URLParams.Add("chirpID", tt.chirpIDParam)
			ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)
			
			if tt.mockGetBearerToken != nil {
				token, err := tt.mockGetBearerToken(http.Header{}) 
				if err == nil && token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
			}
			
			rr := httptest.NewRecorder()
			cfgToPass.handlerChirpDelete(rr, req) 
			if rr.Code != tt.expectedStatusCode { t.Errorf("Status: got %d, want %d. Path: %s, Body: %s", rr.Code, tt.expectedStatusCode, reqPath, rr.Body.String()) }
		})
	}
}
