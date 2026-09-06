package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflags/internal/store"
)

func seed(t *testing.T, s *store.Store, key string, enabled bool, rolloutPercent int) {
	t.Helper()
	if _, err := s.Create(key, enabled, "", rolloutPercent); err != nil {
		t.Fatalf("seed %s: %v", key, err)
	}
}

func doEvaluate(t *testing.T, s *store.Store, key, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/flags/"+key+"/evaluate", nil)
	if user != "" {
		q := req.URL.Query()
		q.Set("user", user)
		req.URL.RawQuery = q.Encode()
	}
	req.SetPathValue("key", key)
	rec := httptest.NewRecorder()
	EvaluateFlag(s, rec, req)
	return rec
}

func decodeResult(t *testing.T, rec *httptest.ResponseRecorder) bool {
	t.Helper()
	var body map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rec.Body.String(), err)
	}
	return body["result"]
}

func TestEvaluateFlagRolloutZero(t *testing.T) {
	s := store.New()
	seed(t, s, "flag0", true, 0)

	rec := doEvaluate(t, s, "flag0", "u1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if decodeResult(t, rec) {
		t.Fatal("expected false for rollout 0")
	}
}

func TestEvaluateFlagRolloutHundred(t *testing.T) {
	s := store.New()
	seed(t, s, "flag100", true, 100)

	rec := doEvaluate(t, s, "flag100", "u1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !decodeResult(t, rec) {
		t.Fatal("expected true for rollout 100")
	}
}

func TestEvaluateFlagDisabled(t *testing.T) {
	s := store.New()
	seed(t, s, "flagoff", false, 100)

	rec := doEvaluate(t, s, "flagoff", "u1")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if decodeResult(t, rec) {
		t.Fatal("expected false when flag is disabled")
	}
}

func TestEvaluateFlagMissingUser(t *testing.T) {
	s := store.New()
	seed(t, s, "flag", true, 100)

	rec := doEvaluate(t, s, "flag", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestEvaluateFlagEmptyUser(t *testing.T) {
	s := store.New()
	seed(t, s, "flag", true, 100)

	req := httptest.NewRequest(http.MethodGet, "/flags/flag/evaluate?user=", nil)
	req.SetPathValue("key", "flag")
	rec := httptest.NewRecorder()
	EvaluateFlag(s, rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestEvaluateFlagUnknownFlag(t *testing.T) {
	s := store.New()

	rec := doEvaluate(t, s, "missing", "u1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
