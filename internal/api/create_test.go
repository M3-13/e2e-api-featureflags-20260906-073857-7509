package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func doCreate(s *store.Store, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
	rec := httptest.NewRecorder()
	CreateFlag(s, rec, req)
	return rec
}

func TestCreateFlagSuccess(t *testing.T) {
	s := store.New()
	rec := doCreate(s, `{"key":"feature-a","enabled":true,"description":"desc","rollout_percent":50}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}

	f, err := s.Get("feature-a")
	if err != nil {
		t.Fatalf("store.Get returned error: %v", err)
	}
	if !f.Enabled || f.Description != "desc" || f.RolloutPercent != 50 {
		t.Errorf("unexpected stored flag: %+v", f)
	}
}

func TestCreateFlagDefaultRolloutPercent(t *testing.T) {
	s := store.New()
	rec := doCreate(s, `{"key":"feature-b","enabled":false}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	f, _ := s.Get("feature-b")
	if f.RolloutPercent != 0 {
		t.Errorf("expected default rollout_percent 0, got %d", f.RolloutPercent)
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	s := store.New()
	_ = doCreate(s, `{"key":"dup","enabled":true}`)

	rec := doCreate(s, `{"key":"dup","enabled":false}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Errorf("expected error object in body, got %q", rec.Body.String())
	}
}

func TestCreateFlagBrokenJSON(t *testing.T) {
	s := store.New()
	rec := doCreate(s, `{"key":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Errorf("expected error object in body, got %q", rec.Body.String())
	}
}

func TestCreateFlagEmptyKey(t *testing.T) {
	s := store.New()
	rec := doCreate(s, `{"key":"","enabled":true}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreateFlagRolloutOutOfRange(t *testing.T) {
	s := store.New()

	for _, body := range []string{
		`{"key":"a","enabled":true,"rollout_percent":-1}`,
		`{"key":"b","enabled":true,"rollout_percent":101}`,
	} {
		rec := doCreate(s, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", body, rec.Code)
		}
	}
}
