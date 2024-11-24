package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

func handlerChirpsValidate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
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

	words := strings.Split(params.Body, " ")
	var filtered []string
	for _, word := range words {
		lowerCase := strings.ToLower(word)
		if strings.Contains(lowerCase, "kerfuffle") || strings.Contains(lowerCase, "sharbert") || strings.Contains(lowerCase, "fornax") {
			filtered = append(filtered, "****")
		} else {
			filtered = append(filtered, word)
		}
	}
	filteredWords := strings.Join(filtered, " ")

	respondWithJSON(w, http.StatusOK, returnVals{CleanedBody: filteredWords})
}
