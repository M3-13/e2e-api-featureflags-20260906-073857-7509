package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflags/internal/store"
)

func TestDeleteFlag(t *testing.T) {
	s := store.New()
	if _, err := s.Create("feature-a", true, "desc", 50); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/flags/feature-a", nil)
	req.SetPathValue("key", "feature-a")
	rr := httptest.NewRecorder()

	DeleteFlag(s, rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rr.Body.String())
	}
}

func TestDeleteFlagNotFound(t *testing.T) {
	s := store.New()

	req := httptest.NewRequest(http.MethodDelete, "/flags/missing", nil)
	req.SetPathValue("key", "missing")
	rr := httptest.NewRecorder()

	DeleteFlag(s, rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Errorf("expected non-empty error field, got %q", rr.Body.String())
	}
}
