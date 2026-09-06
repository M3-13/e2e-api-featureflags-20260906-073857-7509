package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"featureflags/internal/store"
)

func TestListFlagsEmpty(t *testing.T) {
	s := store.New()
	req := httptest.NewRequest(http.MethodGet, "/flags", nil)
	rec := httptest.NewRecorder()

	ListFlags(s, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var flags []store.Flag
	if err := json.Unmarshal(rec.Body.Bytes(), &flags); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if flags == nil {
		t.Fatal("expected empty array, got null")
	}
	if len(flags) != 0 {
		t.Fatalf("expected 0 flags, got %d", len(flags))
	}
}

func TestSortFlagsByKey(t *testing.T) {
	flags := []store.Flag{
		{Key: "zeta"},
		{Key: "alpha"},
		{Key: "mike"},
	}
	sortFlagsByKey(flags)

	want := []string{"alpha", "mike", "zeta"}
	for i, k := range want {
		if flags[i].Key != k {
			t.Fatalf("flag %d: expected key %q, got %q", i, k, flags[i].Key)
		}
	}
}

func TestListFlagsFields(t *testing.T) {
	flags := []store.Flag{
		{ID: 1, Key: "feature-x", Enabled: true, Description: "some description", RolloutPercent: 42},
	}
	sortFlagsByKey(flags)

	rec := httptest.NewRecorder()
	WriteJSON(rec, http.StatusOK, flags)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(decoded))
	}
	f := decoded[0]
	for _, key := range []string{"id", "key", "enabled", "description", "rollout_percent"} {
		if _, ok := f[key]; !ok {
			t.Errorf("missing field %q in %v", key, f)
		}
	}
	if f["id"] != float64(1) {
		t.Errorf("expected id 1, got %v", f["id"])
	}
	if f["key"] != "feature-x" {
		t.Errorf("expected key %q, got %v", "feature-x", f["key"])
	}
	if f["enabled"] != true {
		t.Errorf("expected enabled true, got %v", f["enabled"])
	}
	if f["description"] != "some description" {
		t.Errorf("expected description %q, got %v", "some description", f["description"])
	}
	if f["rollout_percent"] != float64(42) {
		t.Errorf("expected rollout_percent 42, got %v", f["rollout_percent"])
	}
}
