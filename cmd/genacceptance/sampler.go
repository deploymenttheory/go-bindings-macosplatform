//go:build darwin

package main

import (
	"math/rand"
)

// stratifiedSample selects up to n records using round-robin across framework
// buckets, ensuring every framework contributes proportionally. Records within
// each bucket are shuffled by rng before selection.
func stratifiedSample(records []FuncRecord, n int, rng *rand.Rand) []FuncRecord {
	// Group into per-framework buckets.
	order := []string{}
	seen := map[string]bool{}
	buckets := map[string][]FuncRecord{}
	for _, r := range records {
		if !seen[r.Framework] {
			seen[r.Framework] = true
			order = append(order, r.Framework)
		}
		buckets[r.Framework] = append(buckets[r.Framework], r)
	}

	// Shuffle each bucket independently.
	for _, fw := range order {
		b := buckets[fw]
		rng.Shuffle(len(b), func(i, j int) { b[i], b[j] = b[j], b[i] })
		buckets[fw] = b
	}

	// Shuffle the framework order so the round-robin itself is random.
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

	// Round-robin across frameworks until we have n or exhaust all buckets.
	var result []FuncRecord
	for len(result) < n {
		added := 0
		for _, fw := range order {
			if len(result) >= n {
				break
			}
			if len(buckets[fw]) > 0 {
				result = append(result, buckets[fw][0])
				buckets[fw] = buckets[fw][1:]
				added++
			}
		}
		if added == 0 {
			break
		}
	}
	return result
}
