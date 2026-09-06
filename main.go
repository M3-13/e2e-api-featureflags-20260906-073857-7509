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
	if apiKey := os.Getenv("FLAG_API_KEY"); apiKey != "" {
		handler = requireAPIKeyOnFlags(apiKey, handler)
	}
	handler = middleware.Logging(handler)
	return handler
}

// requireAPIKeyOnFlags applies RequireAPIKey only to the /flags routes, leaving
// GET /healthz reachable without authentication.
func requireAPIKeyOnFlags(apiKey string, next http.Handler) http.Handler {
	require := middleware.RequireAPIKey(apiKey)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isFlagsPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		require(next).ServeHTTP(w, r)
	})
}

func isFlagsPath(p string) bool {
	trimmed := strings.TrimPrefix(p, "/")
	seg := trimmed
	if i := strings.IndexByte(trimmed, '/'); i >= 0 {
		seg = trimmed[:i]
	}
	return seg == "flags"
}

func main() {
	s := store.New()
	handler := newHandler(s)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	tlsEnabled := certFile != "" && keyFile != ""

	// Default to loopback so the service never exposes itself without TLS.
	addr := "127.0.0.1:" + port
	if listenAddr := os.Getenv("LISTEN_ADDR"); listenAddr != "" {
		if bindsAllInterfaces(listenAddr) && !tlsEnabled {
			log.Printf("LISTEN_ADDR %q binds all interfaces but TLS is not configured; binding %q instead", listenAddr, addr)
		} else {
			addr = listenAddr
		}
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	if tlsEnabled {
		log.Printf("listening on %s (TLS)", addr)
		if err := srv.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
		return
	}

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

// bindsAllInterfaces reports whether addr's host part denotes every interface
// ("", "0.0.0.0" or "::"), as opposed to a single loopback or specific host.
func bindsAllInterfaces(addr string) bool {
	host := addr
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host = addr[:i]
	}
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	return host == "" || host == "0.0.0.0" || host == "::"
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
