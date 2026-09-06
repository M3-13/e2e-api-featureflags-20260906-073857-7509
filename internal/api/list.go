package api

import (
	"net/http"

	"featureflags/internal/store"
)

func ListFlags(s *store.Store, w http.ResponseWriter, r *http.Request) {
	WriteError(w, http.StatusNotImplemented, "not implemented")
}
