package main

import (
	"encoding/json"
	"github.com/acramatte/Chirpy/internal/database"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"time"
)

func (cfg *apiConfig) handlerChirps(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string    `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}
	type response struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Body      string    `json:"body"`
		UserID    uuid.UUID `json:"user_id"`
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	params := parameters{}
	err := decoder.Decode(&params)
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
		UserID: params.UserID,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create chirp", err)
	}

	respondWithJSON(w, http.StatusCreated, response{ID: chirp.ID, CreatedAt: chirp.CreatedAt, UpdatedAt: chirp.UpdatedAt, Body: chirp.Body, UserID: chirp.UserID})
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
