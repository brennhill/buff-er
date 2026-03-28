package timing

const minSamples = 3

// Estimate holds a duration estimate for a command pattern.
type Estimate struct {
	AvgMinutes float64
	P75Minutes float64
	Samples    int
	Confident  bool // true if we have enough samples
}

// EstimateDuration returns a duration estimate for a command pattern.
func EstimateDuration(store *Store, pattern string) (*Estimate, error) {
	stats, err := store.QueryStats(pattern)
	if err != nil {
		return nil, err
	}

	if stats.Count == 0 {
		return &Estimate{Confident: false, Samples: 0}, nil
	}

	est := &Estimate{
		AvgMinutes: float64(stats.AvgMs) / 60000.0,
		P75Minutes: float64(stats.P75Ms) / 60000.0,
		Samples:    stats.Count,
		Confident:  stats.Count >= minSamples,
	}

	return est, nil
}
