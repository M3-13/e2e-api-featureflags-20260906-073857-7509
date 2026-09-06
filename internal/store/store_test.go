package store

import (
	"errors"
	"sync"
	"testing"
)

func TestCreate(t *testing.T) {
	s := New()

	f, err := s.Create("feature-a", true, "desc", 50)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if f.ID != 1 {
		t.Errorf("expected ID 1, got %d", f.ID)
	}
	if f.Key != "feature-a" || !f.Enabled || f.Description != "desc" || f.RolloutPercent != 50 {
		t.Errorf("unexpected flag contents: %+v", f)
	}

	f2, err := s.Create("feature-b", false, "", 0)
	if err != nil {
		t.Fatalf("second Create returned error: %v", err)
	}
	if f2.ID != 2 {
		t.Errorf("expected ID 2, got %d", f2.ID)
	}
}

func TestCreateConflict(t *testing.T) {
	s := New()
	if _, err := s.Create("dup", true, "", 0); err != nil {
		t.Fatalf("initial Create returned error: %v", err)
	}
	if _, err := s.Create("dup", false, "", 100); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestGet(t *testing.T) {
	s := New()
	_, _ = s.Create("feature-a", true, "desc", 10)

	f, err := s.Get("feature-a")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if f.Key != "feature-a" {
		t.Errorf("expected key feature-a, got %q", f.Key)
	}

	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListSorted(t *testing.T) {
	s := New()
	_, _ = s.Create("bravo", true, "", 0)
	_, _ = s.Create("alpha", true, "", 0)
	_, _ = s.Create("charlie", true, "", 0)

	flags := s.List()
	if len(flags) != 3 {
		t.Fatalf("expected 3 flags, got %d", len(flags))
	}
	want := []string{"alpha", "bravo", "charlie"}
	for i, f := range flags {
		if f.Key != want[i] {
			t.Errorf("expected key %q at %d, got %q", want[i], i, f.Key)
		}
	}
}

func TestUpdate(t *testing.T) {
	s := New()
	_, _ = s.Create("feature-a", true, "original", 10)

	enabled := false
	desc := "changed"
	f, err := s.Update("feature-a", &enabled, &desc, nil)
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if f.Enabled != false {
		t.Errorf("expected Enabled false, got %v", f.Enabled)
	}
	if f.Description != "changed" {
		t.Errorf("expected Description changed, got %q", f.Description)
	}
	if f.RolloutPercent != 10 {
		t.Errorf("expected RolloutPercent unchanged at 10, got %d", f.RolloutPercent)
	}

	got, _ := s.Get("feature-a")
	if got.Enabled || got.Description != "changed" || got.RolloutPercent != 10 {
		t.Errorf("store not persisted after update: %+v", got)
	}
}

func TestUpdateNotFound(t *testing.T) {
	s := New()
	if _, err := s.Update("missing", nil, nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete(t *testing.T) {
	s := New()
	_, _ = s.Create("feature-a", true, "", 0)

	if err := s.Delete("feature-a"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := s.Get("feature-a"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	if err := s.Delete("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing key, got %v", err)
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(n int) {
			defer wg.Done()
			key := "feature-" + string(rune('a'+n%26))
			_, _ = s.Create(key, true, "", 0)
			_, _ = s.Get(key)
			_ = s.List()
			_, _ = s.Update(key, nil, nil, nil)
			_ = s.Delete(key)
		}(i)
	}
	wg.Wait()

	if len(s.List()) != 0 {
		t.Errorf("expected all flags deleted, got %d", len(s.List()))
	}
}
