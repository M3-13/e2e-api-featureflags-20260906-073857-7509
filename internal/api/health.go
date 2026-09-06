package api

import (
	"net/http"
	"runtime"
	"runtime/debug"
)

func Health(w http.ResponseWriter, r *http.Request) {
	body := map[string]string{
		"status":     "ok",
		"go_version": runtime.Version(),
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		body["module_version"] = info.Main.Version
	}

	WriteJSON(w, http.StatusOK, body)
}
