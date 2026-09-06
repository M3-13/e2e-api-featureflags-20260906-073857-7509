package api

import (
	"net/http"
	"sort"

	"featureflags/internal/store"
)

func ListFlags(s *store.Store, w http.ResponseWriter, r *http.Request) {
	flags := s.List()
	sortFlagsByKey(flags)
	if flags == nil {
		flags = []store.Flag{}
	}
	WriteJSON(w, http.StatusOK, flags)
}

func sortFlagsByKey(flags []store.Flag) {
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Key < flags[j].Key
	})
}
