package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"

	"featureflags/internal/store"
)

var keyPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,128}$`)

const maxDescriptionLength = 1024

type createRequest struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent *int   `json:"rollout_percent"`
}

func CreateFlag(s *store.Store, w http.ResponseWriter, r *http.Request) {
	var req createRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		WriteError(w, http.StatusBadRequest, "unexpected trailing data")
		return
	}

	if !keyPattern.MatchString(req.Key) {
		WriteError(w, http.StatusBadRequest, "invalid key")
		return
	}

	if len(req.Description) > maxDescriptionLength {
		WriteError(w, http.StatusBadRequest, "description too long")
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
		if errors.Is(err, store.ErrConflict) {
			WriteError(w, http.StatusConflict, err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	WriteJSON(w, http.StatusCreated, flag)
}
