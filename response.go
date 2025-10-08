package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func respondWithJSON(w http.ResponseWriter, payload interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Could not even marshal the json: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.WriteHeader(status)
	w.Write(dat)
}

func respondWithError(w http.ResponseWriter, error string, status int) {
	respondWithJSON(w, ErrorResponse{Error: error}, status)
}
