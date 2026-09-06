package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func TestHealthzOpenWithAPIKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")
	h := newHandler(store.New())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected GET /healthz 200 without token, got %d", rec.Code)
	}
}

func TestFlagsRequireAPIKey(t *testing.T) {
	t.Setenv("FLAG_API_KEY", "secret")

	t.Run("missing token returns 401", func(t *testing.T) {
		h := newHandler(store.New())
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/flags",
			strings.NewReader(`{"key":"a","enabled":true}`)))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"unauthorized"`) {
			t.Fatalf("expected unauthorized JSON error, got %q", rec.Body.String())
		}
	})

	t.Run("wrong token returns 401", func(t *testing.T) {
		h := newHandler(store.New())
		req := httptest.NewRequest(http.MethodPost, "/flags",
			strings.NewReader(`{"key":"a","enabled":true}`))
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("correct bearer token creates flag", func(t *testing.T) {
		h := newHandler(store.New())
		req := httptest.NewRequest(http.MethodPost, "/flags",
			strings.NewReader(`{"key":"a","enabled":true,"rollout_percent":50}`))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestBindsAllInterfaces(t *testing.T) {
	cases := []struct {
		addr string
		want bool
	}{
		{":8080", true},
		{"0.0.0.0:8080", true},
		{"[::]:8080", true},
		{"127.0.0.1:8080", false},
		{"localhost:8080", false},
	}
	for _, c := range cases {
		if got := bindsAllInterfaces(c.addr); got != c.want {
			t.Errorf("bindsAllInterfaces(%q) = %v, want %v", c.addr, got, c.want)
		}
	}
}
