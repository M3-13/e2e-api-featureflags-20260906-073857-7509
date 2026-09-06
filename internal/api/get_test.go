package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflags/internal/store"
)

func TestGetFlagExisting(t *testing.T) {
	s := store.New()
	_, err := s.Create("feature-x", true, "some description", 50)
	if err != nil {
		t.Fatalf("seed flag: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/flags/feature-x", nil)
	req.SetPathValue("key", "feature-x")
	rec := httptest.NewRecorder()

	GetFlag(s, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var flag store.Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if flag.Key != "feature-x" {
		t.Fatalf("expected key feature-x, got %q", flag.Key)
	}
	if !flag.Enabled {
		t.Fatalf("expected enabled true")
	}
	if flag.RolloutPercent != 50 {
		t.Fatalf("expected rollout_percent 50, got %d", flag.RolloutPercent)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	s := store.New()

	req := httptest.NewRequest(http.MethodGet, "/flags/missing", nil)
	req.SetPathValue("key", "missing")
	rec := httptest.NewRecorder()

	GetFlag(s, rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected non-empty error message")
	}
}
