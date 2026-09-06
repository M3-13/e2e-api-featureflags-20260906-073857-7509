package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func TestRoutesAreWired(t *testing.T) {
	h := newHandler(store.New())
	cases := []struct{ method, path string }{
		{"POST", "/flags"},
		{"GET", "/flags"},
		{"GET", "/flags/x"},
		{"PUT", "/flags/x"},
		{"DELETE", "/flags/x"},
		{"GET", "/flags/x/evaluate"},
		{"GET", "/healthz"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code == http.StatusNotFound && !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s %s -> 404 (route not wired)", c.method, c.path)
		}
	}
}

func TestWrongMethodReturnsJSON405(t *testing.T) {
	h := newHandler(store.New())
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("405 body not JSON error object: %q", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "Method Not Allowed") {
		t.Fatalf("405 body is ServeMux text default: %q", rec.Body.String())
	}
}
