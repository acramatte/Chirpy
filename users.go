package main

import (
	"encoding/json"
	"github.com/acramatte/Chirpy/internal/auth"
	"github.com/acramatte/Chirpy/internal/database"
	"github.com/google/uuid"
	"net/http"
	"time"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Password     string    `json:"password"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

func (cfg *apiConfig) handlerUsersCreation(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}
	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{params.Email, hashedPassword})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create user", err)
	}

	respondWithJSON(w, http.StatusCreated, User{ID: user.ID, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Email: user.Email})
}
