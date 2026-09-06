package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"featureflags/internal/store"
)

func doUpdate(t *testing.T, s *store.Store, key, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPut, "/flags/"+key, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("key", key)
	rec := httptest.NewRecorder()
	UpdateFlag(s, rec, req)
	return rec
}

func TestUpdateFlagChangesExistingFlag(t *testing.T) {
	s := store.New()
	if _, err := s.Create("foo", true, "desc", 50); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	rec := doUpdate(t, s, "foo", `{"enabled":false,"description":"updated","rollout_percent":80}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var flag store.Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if flag.Key != "foo" {
		t.Errorf("key = %q, want foo", flag.Key)
	}
	if flag.Enabled {
		t.Errorf("enabled = true, want false")
	}
	if flag.Description != "updated" {
		t.Errorf("description = %q, want updated", flag.Description)
	}
	if flag.RolloutPercent != 80 {
		t.Errorf("rollout_percent = %d, want 80", flag.RolloutPercent)
	}
}

func TestUpdateFlagMissingFieldsUnchanged(t *testing.T) {
	s := store.New()
	if _, err := s.Create("foo", true, "desc", 50); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	rec := doUpdate(t, s, "foo", `{"description":"new desc"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var flag store.Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if !flag.Enabled {
		t.Errorf("enabled = false, want true (unchanged)")
	}
	if flag.Description != "new desc" {
		t.Errorf("description = %q, want new desc", flag.Description)
	}
	if flag.RolloutPercent != 50 {
		t.Errorf("rollout_percent = %d, want 50 (unchanged)", flag.RolloutPercent)
	}
}

func TestUpdateFlagUnknownKeyReturns404(t *testing.T) {
	s := store.New()

	rec := doUpdate(t, s, "missing", `{"enabled":true}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("expected non-empty error message, got %q", body["error"])
	}
}

func TestUpdateFlagRolloutOutOfRangeReturns400(t *testing.T) {
	s := store.New()
	if _, err := s.Create("foo", true, "desc", 50); err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	for _, pct := range []int{-1, 101} {
		body := `{"rollout_percent":` + strconv.Itoa(pct) + `}`
		rec := doUpdate(t, s, "foo", body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("rollout_percent=%d: expected 400, got %d", pct, rec.Code)
		}
	}
}

func TestUpdateFlagInvalidJSONReturns400(t *testing.T) {
	s := store.New()

	rec := doUpdate(t, s, "foo", `{"enabled":`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("expected non-empty error message, got %q", body["error"])
	}
}
