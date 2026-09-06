package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingDoesNotLogQuery(t *testing.T) {
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/flags/evaluate?user=secret123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	out := buf.String()
	if strings.Contains(out, "secret123") {
		t.Fatalf("log output leaks query string: %q", out)
	}
	if !strings.Contains(out, http.MethodGet) {
		t.Fatalf("log output missing method: %q", out)
	}
	if !strings.Contains(out, "/flags/evaluate") {
		t.Fatalf("log output missing path: %q", out)
	}
	if !strings.Contains(out, "200") {
		t.Fatalf("log output missing status: %q", out)
	}
}

func TestLimitBodyRejectsOversized(t *testing.T) {
	handler := LimitBody(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/flags", bytes.NewReader([]byte("1234567890")))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Fatalf("expected JSON error body, got %q", rec.Body.String())
	}
}

func TestLimitBodyAllowsWithinLimit(t *testing.T) {
	var got []byte
	handler := LimitBody(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodPost, "/flags", bytes.NewReader([]byte("hello")))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if string(got) != "hello" {
		t.Fatalf("body not passed through: %q", got)
	}
}
