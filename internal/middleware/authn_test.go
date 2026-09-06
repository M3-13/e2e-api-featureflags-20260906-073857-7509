package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func requireOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRequireAPIKeyValidBearerToken(t *testing.T) {
	h := RequireAPIKey("secret")(requireOKHandler())
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAPIKeyValidXAPIKey(t *testing.T) {
	h := RequireAPIKey("secret")(requireOKHandler())
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("X-API-Key", "secret")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAPIKeyWrongToken(t *testing.T) {
	h := RequireAPIKey("secret")(requireOKHandler())
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"unauthorized"`) {
		t.Fatalf("expected unauthorized JSON error, got %q", rec.Body.String())
	}
}

func TestRequireAPIKeyMissingToken(t *testing.T) {
	h := RequireAPIKey("secret")(requireOKHandler())
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"unauthorized"`) {
		t.Fatalf("expected unauthorized JSON error, got %q", rec.Body.String())
	}
}

// TestRequireAPIKeyEmptyKeyLeavesServiceOpen documents the "open" mode used
// when FLAG_API_KEY is unset: with an empty key the middleware lets every
// request through, so GET /healthz (and everything else) stays reachable
// without a token.
func TestRequireAPIKeyEmptyKeyLeavesServiceOpen(t *testing.T) {
	h := RequireAPIKey("")(requireOKHandler())
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for open mode, got %d", rec.Code)
	}
}
