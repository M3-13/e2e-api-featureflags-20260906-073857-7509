package evaluate

import (
	"strconv"
	"testing"
)

func TestDecideDeterministic(t *testing.T) {
	first := Decide("myflag", "user-42", 37)
	for i := 0; i < 100; i++ {
		if got := Decide("myflag", "user-42", 37); got != first {
			t.Fatalf("decision not stable across calls: first=%v, call %d=%v", first, i, got)
		}
	}
}

func TestDecideZeroPercent(t *testing.T) {
	for _, user := range []string{"a", "b", "c", "user-42", "x"} {
		if Decide("flag", user, 0) {
			t.Fatalf("expected false for rollout 0, user %q", user)
		}
	}
}

func TestDecideHundredPercent(t *testing.T) {
	for _, user := range []string{"a", "b", "c", "user-42", "x"} {
		if !Decide("flag", user, 100) {
			t.Fatalf("expected true for rollout 100, user %q", user)
		}
	}
}

func TestDecideDistributionAcrossUsers(t *testing.T) {
	const users = 1000
	trueCount := 0
	for i := 0; i < users; i++ {
		if Decide("flag", "user"+strconv.Itoa(i), 50) {
			trueCount++
		}
	}
	if trueCount == 0 || trueCount == users {
		t.Fatalf("expected a mix of decisions across users, got %d/%d true", trueCount, users)
	}
	if trueCount < 300 || trueCount > 700 {
		t.Fatalf("rollout 50 should distribute roughly evenly, got %d/%d true", trueCount, users)
	}
}
