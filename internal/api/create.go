package api

import (
	"encoding/json"
	"net/http"

	"featureflags/internal/store"
)

type createRequest struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent *int   `json:"rollout_percent"`
}

func CreateFlag(s *store.Store, w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		WriteError(w, http.StatusBadRequest, "key must not be empty")
		return
	}

	rolloutPercent := 0
	if req.RolloutPercent != nil {
		rolloutPercent = *req.RolloutPercent
	}
	if rolloutPercent < 0 || rolloutPercent > 100 {
		WriteError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}

	flag, err := s.Create(req.Key, req.Enabled, req.Description, rolloutPercent)
	if err != nil {
		if err == store.ErrConflict {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	WriteJSON(w, http.StatusCreated, flag)
}
