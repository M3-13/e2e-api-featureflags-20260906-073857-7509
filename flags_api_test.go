package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"featureflags/internal/store"
)

// createFlagPOST performs an end-to-end POST /flags against the given handler
// and returns the recorded response.
func createFlagPOST(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/flags", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// evaluateGET performs an end-to-end GET /flags/{key}/evaluate?user={id} and
// returns the recorded response.
func evaluateGET(t *testing.T, h http.Handler, key, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/flags/"+key+"/evaluate?user="+user, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestEndpointsWired verifies that every declared route is actually wired.
// A flag is created first via POST /flags so the key-scoped routes
// (GET/PUT/DELETE/evaluate) exercise their real, expected responses for an
// existing flag. A 404 here is NOT a wiring error: the spec requires 404 for
// unknown keys (AC-04, AC-06), so wiring is proven by the expected success
// status (200/204) and not by the absence of a 404.
func TestEndpointsWired(t *testing.T) {
	s := store.New()
	h := newHandler(s)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/flags",
		strings.NewReader(`{"key":"wired-key","enabled":true,"rollout_percent":50}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /flags -> %d (want 201): %s", rec.Code, rec.Body.String())
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{name: "list", method: http.MethodGet, path: "/flags", want: http.StatusOK},
		{name: "get", method: http.MethodGet, path: "/flags/wired-key", want: http.StatusOK},
		{name: "update", method: http.MethodPut, path: "/flags/wired-key", body: `{"enabled":false}`, want: http.StatusOK},
		{name: "evaluate", method: http.MethodGet, path: "/flags/wired-key/evaluate?user=alice", want: http.StatusOK},
		{name: "delete", method: http.MethodDelete, path: "/flags/wired-key", want: http.StatusNoContent},
		{name: "health", method: http.MethodGet, path: "/healthz", want: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, body))
			if rec.Code != tc.want {
				t.Errorf("%s %s -> %d (want %d): %s", tc.method, tc.path, rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

// TestEvaluateRolloutBoundaries verifies AC-08: rollout_percent=0 always
// yields "aus", rollout_percent=100 always yields "an", and enabled=false
// always yields "aus". Each boundary case runs against a fresh store and
// handler so that no key collides with a flag created by a sibling case.
func TestEvaluateRolloutBoundaries(t *testing.T) {
	cases := []struct {
		name           string
		key            string
		enabled        bool
		rolloutPercent int
		wantResult     bool
	}{
		{name: "rollout 0", key: "boundary-zero", enabled: true, rolloutPercent: 0, wantResult: false},
		{name: "rollout 100", key: "boundary-hundred", enabled: true, rolloutPercent: 100, wantResult: true},
		{name: "disabled", key: "boundary-disabled", enabled: false, rolloutPercent: 100, wantResult: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := store.New()
			h := newHandler(s)

			body, err := json.Marshal(map[string]any{
				"key":             tc.key,
				"enabled":         tc.enabled,
				"rollout_percent": tc.rolloutPercent,
			})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}

			createRec := createFlagPOST(t, h, string(body))
			if createRec.Code != http.StatusCreated {
				t.Fatalf("create %s -> %d (want 201): %s", tc.name, createRec.Code, createRec.Body.String())
			}

			evalRec := evaluateGET(t, h, tc.key, "alice")
			if evalRec.Code != http.StatusOK {
				t.Fatalf("evaluate %s -> %d (want 200): %s", tc.name, evalRec.Code, evalRec.Body.String())
			}

			var result struct {
				Result bool `json:"result"`
			}
			if err := json.Unmarshal(evalRec.Body.Bytes(), &result); err != nil {
				t.Fatalf("invalid evaluate JSON %q: %v", evalRec.Body.String(), err)
			}
			if result.Result != tc.wantResult {
				t.Fatalf("evaluate %s = %v, want %v", tc.name, result.Result, tc.wantResult)
			}
		})
	}
}
