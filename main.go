package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"featureflags/internal/api"
	"featureflags/internal/middleware"
	"featureflags/internal/store"
)

const oneMiB = 1 << 20

func storeHandler(s *store.Store, fn func(*store.Store, http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(s, w, r)
	})
}

func newHandler(s *store.Store) http.Handler {
	limit := middleware.LimitBody(oneMiB)

	mux := http.NewServeMux()
	mux.Handle("POST /flags", limit(storeHandler(s, api.CreateFlag)))
	mux.Handle("GET /flags", storeHandler(s, api.ListFlags))
	mux.Handle("GET /flags/{key}", storeHandler(s, api.GetFlag))
	mux.Handle("PUT /flags/{key}", limit(storeHandler(s, api.UpdateFlag)))
	mux.Handle("DELETE /flags/{key}", storeHandler(s, api.DeleteFlag))
	mux.Handle("GET /flags/{key}/evaluate", storeHandler(s, api.EvaluateFlag))
	mux.Handle("GET /healthz", http.HandlerFunc(api.Health))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if allowed := methodsForPath(r.URL.Path); allowed != nil && !allowed[r.Method] {
			api.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		api.WriteError(w, http.StatusNotFound, "not found")
	})

	var handler http.Handler = mux
	handler = json405(handler)
	handler = middleware.Logging(handler)
	return handler
}

func main() {
	s := store.New()
	handler := newHandler(s)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	log.Printf("listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

type methodNotAllowedWriter struct {
	http.ResponseWriter
	handled bool
}

func (w *methodNotAllowedWriter) WriteHeader(code int) {
	if code == http.StatusMethodNotAllowed {
		w.handled = true
		api.WriteError(w.ResponseWriter, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *methodNotAllowedWriter) Write(p []byte) (int, error) {
	if w.handled {
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}

func methodsForPath(path string) map[string]bool {
	segs := strings.Split(strings.Trim(path, "/"), "/")
	if len(segs) == 1 && segs[0] == "" {
		return nil
	}
	switch {
	case len(segs) == 1 && segs[0] == "healthz":
		return map[string]bool{http.MethodGet: true}
	case len(segs) == 1 && segs[0] == "flags":
		return map[string]bool{http.MethodGet: true, http.MethodPost: true}
	case len(segs) == 2 && segs[0] == "flags":
		return map[string]bool{http.MethodGet: true, http.MethodPut: true, http.MethodDelete: true}
	case len(segs) == 3 && segs[0] == "flags" && segs[2] == "evaluate":
		return map[string]bool{http.MethodGet: true}
	default:
		return nil
	}
}

func json405(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&methodNotAllowedWriter{ResponseWriter: w}, r)
	})
}
