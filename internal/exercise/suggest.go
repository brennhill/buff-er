package exercise

import (
	"math"
	"math/rand"
)

// Suggest picks a random exercise appropriate for the given wait time.
// It prefers exercises whose duration range contains the wait time.
// Falls back to the closest match if none fit exactly.
func Suggest(catalog []Exercise, waitMinutes float64) *Exercise {
	if len(catalog) == 0 {
		return nil
	}

	// Find exercises that fit within the wait time
	var candidates []Exercise
	for _, e := range catalog {
		if float64(e.MinMinutes) <= waitMinutes && float64(e.MaxMinutes) <= waitMinutes {
			candidates = append(candidates, e)
		}
	}

	if len(candidates) > 0 {
		pick := candidates[rand.Intn(len(candidates))]
		return &pick
	}

	// No exact fit — find the closest exercise by min duration
	var best *Exercise
	bestDist := math.MaxFloat64
	for i := range catalog {
		e := &catalog[i]
		mid := float64(e.MinMinutes+e.MaxMinutes) / 2.0
		dist := math.Abs(mid - waitMinutes)
		// Only suggest exercises that can be completed in the wait time
		if float64(e.MinMinutes) <= waitMinutes && dist < bestDist {
			bestDist = dist
			best = e
		}
	}

	if best != nil {
		result := *best
		return &result
	}

	// Last resort: just pick the shortest exercise
	shortest := catalog[0]
	for _, e := range catalog[1:] {
		if e.MinMinutes < shortest.MinMinutes {
			shortest = e
		}
	}
	return &shortest
}
