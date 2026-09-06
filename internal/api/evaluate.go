package api

import (
	"net/http"

	"featureflags/internal/evaluate"
	"featureflags/internal/store"
)

// EvaluateFlag serves GET /flags/{key}/evaluate?user={id}.
// The user value is used only for the hash computation and is never stored.
func EvaluateFlag(s *store.Store, w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		WriteError(w, http.StatusBadRequest, "user is required")
		return
	}

	key := r.PathValue("key")
	flag, err := s.Get(key)
	if err != nil {
		if err == store.ErrNotFound {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	result := false
	switch {
	case !flag.Enabled:
		result = false
	case flag.RolloutPercent >= 100:
		result = true
	case flag.RolloutPercent > 0:
		result = evaluate.Decide(flag.Key, user, flag.RolloutPercent)
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"result": result})
}
