package main

import (
	"database/sql"
	"github.com/acramatte/Chirpy/internal/auth" // Added for auth functions
	"github.com/acramatte/Chirpy/internal/database"
	"github.com/google/uuid" // Added for uuid.UUID type in function signatures
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log"
	"net/http"
	"os"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits  atomic.Int32
	db              database.Querier // Changed from *database.Queries to database.Querier
	platform        string
	jwtSecret       string
	polkaWebhookKey string

	// New fields for injectable auth functions
	validateJWT    func(tokenString, tokenSecret string) (uuid.UUID, error)
	getBearerToken func(headers http.Header) (string, error)
	getAPIKey      func(headers http.Header) (string, error)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	dbURL := MustEnv("DB_URL")
	platform := MustEnv("PLATFORM")
	jwtSecret := MustEnv("JWT_SECRET")
	polkaWebhookKey := MustEnv("POLKA_KEY")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	dbQueries := database.New(db)

	fs := http.FileServer(http.Dir("."))
	apiCfg := apiConfig{
		fileserverHits:  atomic.Int32{},
		db:              dbQueries,
		platform:        platform,
		jwtSecret:       jwtSecret,
		polkaWebhookKey: polkaWebhookKey,

		// Assign actual auth functions
		validateJWT:    auth.ValidateJWT,
		getBearerToken: auth.GetBearerToken,
		getAPIKey:      auth.GetAPIKey,
	}
	apiCfg.fileserverHits.Store(0)

	serveMux := http.NewServeMux()
	fsHandler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", fs))
	serveMux.Handle("/app/", fsHandler)

	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)

	serveMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

	serveMux.HandleFunc("GET /api/chirps", apiCfg.handlerChirpsRetrieve)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handlerChirpsCreate)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerChirpGet)
	serveMux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerChirpDelete)

	serveMux.HandleFunc("POST /api/users", apiCfg.handlerUsersCreation)
	serveMux.HandleFunc("PUT /api/users", apiCfg.handlerUsersUpdate)

	serveMux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerUpgradeRed)

	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)

	server := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}
	log.Println("Starting server on :8080")
	log.Fatal(server.ListenAndServe())
}

// MustEnv reads an environment variable and terminates immediately if it is missing
func MustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Environment variable %s must be set", key)
	}
	return val
}
