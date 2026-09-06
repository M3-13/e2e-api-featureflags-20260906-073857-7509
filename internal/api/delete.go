package api

import (
	"errors"
	"net/http"

	"featureflags/internal/store"
)

func DeleteFlag(s *store.Store, w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := s.Delete(key); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
