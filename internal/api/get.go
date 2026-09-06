package api

import (
	"errors"
	"net/http"

	"featureflags/internal/store"
)

func GetFlag(s *store.Store, w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	flag, err := s.Get(key)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "flag not found")
			return
		}
		WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, flag)
}
