package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func TestRoutesAreWired(t *testing.T) {
	s := store.New()
	if _, err := s.Create("x", true, "", 0); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	h := newHandler(s)
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

func TestUnknownPathReturnsJSON404(t *testing.T) {
	h := newHandler(store.New())
	for _, path := range []string{"/nope", "/flags/x/y/z", "/other"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))

		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s -> expected 404, got %d", path, rec.Code)
		}
		if !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Errorf("GET %s -> 404 without JSON content-type: %q", path, rec.Header().Get("Content-Type"))
		}
		if !strings.Contains(rec.Body.String(), `"error"`) {
			t.Errorf("GET %s -> body not a JSON error object: %q", path, rec.Body.String())
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
