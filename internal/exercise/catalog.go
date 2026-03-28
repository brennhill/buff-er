package exercise

// Exercise represents a single exercise suggestion.
type Exercise struct {
	Name        string `json:"name" toml:"name"`
	Description string `json:"description" toml:"description"`
	MinMinutes  int    `json:"min_minutes" toml:"min_minutes"`
	MaxMinutes  int    `json:"max_minutes" toml:"max_minutes"`
	Category    string `json:"category" toml:"category"` // stretch, strength, movement
}

// DefaultCatalog returns the built-in exercise list.
func DefaultCatalog() []Exercise {
	return []Exercise{
		// Quick (1-2 min)
		{Name: "Neck rolls", Description: "Slowly roll your head in circles, 5 each direction", MinMinutes: 1, MaxMinutes: 2, Category: "stretch"},
		{Name: "Wrist stretches", Description: "Extend arms, pull fingers back gently, 15s each hand", MinMinutes: 1, MaxMinutes: 2, Category: "stretch"},
		{Name: "Shoulder shrugs", Description: "Raise shoulders to ears, hold 5s, drop. Repeat 10x", MinMinutes: 1, MaxMinutes: 2, Category: "stretch"},
		{Name: "Standing calf raises", Description: "Rise up on your toes, hold 2s, lower. 15 reps", MinMinutes: 1, MaxMinutes: 2, Category: "strength"},
		{Name: "Deep breaths", Description: "4 counts in, hold 4, out 4. Repeat 5x", MinMinutes: 1, MaxMinutes: 2, Category: "movement"},

		// Short (2-4 min)
		{Name: "Desk pushups", Description: "10 pushups against your desk edge. Rest. Repeat.", MinMinutes: 2, MaxMinutes: 4, Category: "strength"},
		{Name: "Wall sit", Description: "Back against wall, knees at 90 degrees. Hold 30s x 3", MinMinutes: 2, MaxMinutes: 4, Category: "strength"},
		{Name: "Hip flexor stretch", Description: "Lunge position, push hips forward. 30s each side x 2", MinMinutes: 2, MaxMinutes: 4, Category: "stretch"},
		{Name: "Squats", Description: "15 bodyweight squats. Rest 30s. Repeat.", MinMinutes: 2, MaxMinutes: 4, Category: "strength"},
		{Name: "Walk around", Description: "Get up and walk around your space. Hydrate.", MinMinutes: 2, MaxMinutes: 4, Category: "movement"},
		{Name: "Torso twists", Description: "Seated or standing, rotate torso left and right. 20 reps", MinMinutes: 2, MaxMinutes: 4, Category: "stretch"},

		// Medium (4-8 min)
		{Name: "Sun salutations", Description: "3 slow sun salutation flows", MinMinutes: 4, MaxMinutes: 8, Category: "stretch"},
		{Name: "Stair walk", Description: "Walk up and down stairs for a few minutes", MinMinutes: 4, MaxMinutes: 8, Category: "movement"},
		{Name: "Plank circuit", Description: "30s front plank, 30s each side. Rest 30s. Repeat.", MinMinutes: 4, MaxMinutes: 8, Category: "strength"},
		{Name: "Full body stretch", Description: "Hamstrings, quads, shoulders, back. 30s each", MinMinutes: 4, MaxMinutes: 8, Category: "stretch"},

		// Long (8-15 min)
		{Name: "Walk outside", Description: "Go outside, walk around the block, get some air", MinMinutes: 8, MaxMinutes: 15, Category: "movement"},
		{Name: "Yoga flow", Description: "A few minutes of gentle yoga — forward folds, twists, hip openers", MinMinutes: 8, MaxMinutes: 15, Category: "stretch"},
		{Name: "Bodyweight circuit", Description: "10 pushups, 15 squats, 30s plank. 3 rounds.", MinMinutes: 8, MaxMinutes: 15, Category: "strength"},
	}
}
