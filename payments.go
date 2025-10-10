package main

import (
	"chirpy/internal/auth"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type PaymentEvent struct {
	Event string `json:"event"`
	Data  struct {
		UserID uuid.UUID `json:"user_id"`
	} `json:"data"`
}

func (cfg *apiConfig) handlePayment(w http.ResponseWriter, req *http.Request) {

	apiKey := auth.GetAPIKey(req.Header)
	if apiKey != cfg.paymentAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := PaymentEvent{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, "invalid payment event", http.StatusBadRequest)
	}

	// ignore any event other than a successful payment
	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = cfg.db.UpgradeUser(req.Context(), params.Data.UserID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
