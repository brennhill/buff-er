package exercise

import (
	"math"
	"math/rand"
)

// Suggest picks a random exercise appropriate for the given wait time.
// It prefers exercises whose max duration fits within the wait time.
// Falls back to the closest match if none fit exactly.
func Suggest(catalog []Exercise, waitMinutes float64) *Exercise {
	if len(catalog) == 0 {
		return nil
	}

	// Find exercises whose max duration fits within the wait time
	if candidates := filterByMaxDuration(catalog, waitMinutes); len(candidates) > 0 {
		return pickRandom(candidates)
	}

	// No exact fit — collect exercises near the best match
	if near := findNearestExercises(catalog, waitMinutes); len(near) > 0 {
		return pickRandom(near)
	}

	// Last resort: pick from shortest exercises
	if shortest := findShortest(catalog); len(shortest) > 0 {
		return pickRandom(shortest)
	}
	return nil
}

func filterByMaxDuration(catalog []Exercise, waitMinutes float64) []Exercise {
	var result []Exercise
	for _, e := range catalog {
		if float64(e.MinMinutes) <= waitMinutes && float64(e.MaxMinutes) <= waitMinutes {
			result = append(result, e)
		}
	}
	return result
}

func findNearestExercises(catalog []Exercise, waitMinutes float64) []Exercise {
	bestDist := math.MaxFloat64
	for _, e := range catalog {
		if float64(e.MinMinutes) > waitMinutes {
			continue
		}
		mid := float64(e.MinMinutes+e.MaxMinutes) / 2.0
		dist := math.Abs(mid - waitMinutes)
		if dist < bestDist {
			bestDist = dist
		}
	}

	if bestDist == math.MaxFloat64 {
		return nil
	}

	tolerance := bestDist + 1.0
	var result []Exercise
	for _, e := range catalog {
		if float64(e.MinMinutes) > waitMinutes {
			continue
		}
		mid := float64(e.MinMinutes+e.MaxMinutes) / 2.0
		if math.Abs(mid-waitMinutes) <= tolerance {
			result = append(result, e)
		}
	}
	return result
}

func findShortest(catalog []Exercise) []Exercise {
	shortest := catalog[0].MinMinutes
	for _, e := range catalog[1:] {
		if e.MinMinutes < shortest {
			shortest = e.MinMinutes
		}
	}
	var result []Exercise
	for _, e := range catalog {
		if e.MinMinutes == shortest {
			result = append(result, e)
		}
	}
	return result
}

func pickRandom(exercises []Exercise) *Exercise {
	pick := exercises[rand.Intn(len(exercises))]
	return &pick
}
