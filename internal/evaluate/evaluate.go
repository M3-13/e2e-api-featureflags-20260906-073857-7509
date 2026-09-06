package evaluate

import "hash/fnv"

// Decide computes a deterministic rollout decision for a (key, user) pair.
// It hashes the concatenation "key:user" with FNV-1a and returns true when
// (hash % 100) < rolloutPercent. rolloutPercent == 0 therefore always yields
// false and rolloutPercent == 100 always yields true.
func Decide(key, user string, rolloutPercent int) bool {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key + ":" + user))
	return int(h.Sum32()%100) < rolloutPercent
}
