package main

import (
	"encoding/json"
	// "github.com/acramatte/Chirpy/internal/auth" // Removed unused import
	"github.com/google/uuid"
	"net/http"
)

func (cfg *apiConfig) handlerUpgradeRed(w http.ResponseWriter, r *http.Request) {
	apiKey, err := cfg.getAPIKey(r.Header) // Replaced auth.GetAPIKey
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find api key", err)
		return
	}
	if apiKey != cfg.polkaWebhookKey {
		respondWithError(w, http.StatusUnauthorized, "API key not authorized", err)
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}
	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	_, err = cfg.db.UpgradeToRed(r.Context(), params.Data.UserID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found", err)
		return
	}
	respondWithJSON(w, http.StatusNoContent, struct{}{})
}
