package main

import (
	"context"
	"encoding/json"
	"fmt"
	// "github.com/acramatte/Chirpy/internal/auth" // Removed unused import
	"github.com/acramatte/Chirpy/internal/database"
	"github.com/go-chi/chi/v5" // Added for chi.URLParam
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) handlerChirpGet(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := chi.URLParam(r, "chirpID") // Changed from r.PathValue
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	dbChirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	}
	respondWithJSON(w, http.StatusOK, Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserID:    dbChirp.UserID,
	})
}

func (cfg *apiConfig) handlerChirpDelete(w http.ResponseWriter, r *http.Request) {
	token, err := cfg.getBearerToken(r.Header) // Replaced auth.GetBearerToken
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := cfg.validateJWT(token, cfg.jwtSecret) // Replaced auth.ValidateJWT
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	chirpIDStr := chi.URLParam(r, "chirpID") // Changed from r.PathValue
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	}

	if chirp.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Not your chirp", err)
		return
	}

	err = cfg.db.DeleteChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Couldn't delete chirp %v", chirpID), err)
		return
	}
	respondWithJSON(w, http.StatusNoContent, struct{}{})
}

func (cfg *apiConfig) handlerChirpsRetrieve(w http.ResponseWriter, r *http.Request) {
	sort := r.URL.Query().Get("sort")
	order := strings.ToLower(sort)
	if order != "asc" && order != "desc" {
		order = "asc" // Default to ASC if invalid
	}

	s := r.URL.Query().Get("author_id")
	dbChirps, err := cfg.getChirps(r.Context(), s, order)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't retrieve chirps", err)
		return
	}

	var chirps []Chirp
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{
			ID:        dbChirp.ID,
			CreatedAt: dbChirp.CreatedAt,
			UpdatedAt: dbChirp.UpdatedAt,
			Body:      dbChirp.Body,
			UserID:    dbChirp.UserID,
		})
	}
	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) getChirps(c context.Context, authorId, sortOrder string) ([]database.Chirp, error) {
	if authorId == "" {
		return cfg.db.GetChirps(c, sortOrder)
	}

	parsedID, err := uuid.Parse(authorId)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse user id: %w", err)
	}

	return cfg.db.GetChirpsByAuthorId(c, database.GetChirpsByAuthorIdParams{
		UserID:  parsedID,
		Column2: sortOrder,
	})
}

func (cfg *apiConfig) handlerChirpsCreate(w http.ResponseWriter, r *http.Request) {
	token, err := cfg.getBearerToken(r.Header) // Replaced auth.GetBearerToken
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}
	userID, err := cfg.validateJWT(token, cfg.jwtSecret) // Replaced auth.ValidateJWT
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	type parameters struct {
		Body string `json:"body"`
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}
	filteredBody := getCleanedBody(params.Body)

	chirp, err := cfg.db.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   filteredBody,
		UserID: userID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create chirp", err)
	}

	respondWithJSON(w, http.StatusCreated, Chirp{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID})
}

func getCleanedBody(body string) string {
	words := strings.Split(body, " ")
	var filtered []string
	for _, word := range words {
		lowerCase := strings.ToLower(word)
		if strings.Contains(lowerCase, "kerfuffle") || strings.Contains(lowerCase, "sharbert") || strings.Contains(lowerCase, "fornax") {
			filtered = append(filtered, "****")
		} else {
			filtered = append(filtered, word)
		}
	}
	return strings.Join(filtered, " ")
}
