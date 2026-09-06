package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"featureflags/internal/store"
)

type updateFlagRequest struct {
	Enabled        *bool   `json:"enabled"`
	Description    *string `json:"description"`
	RolloutPercent *int    `json:"rollout_percent"`
}

func UpdateFlag(s *store.Store, w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		WriteError(w, http.StatusBadRequest, "key must not be empty")
		return
	}

	var req updateFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.RolloutPercent != nil && (*req.RolloutPercent < 0 || *req.RolloutPercent > 100) {
		WriteError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}

	flag, err := s.Update(key, req.Enabled, req.Description, req.RolloutPercent)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	WriteJSON(w, http.StatusOK, flag)
}
