package exercise

import "testing"

func TestSuggestFindsExactMatch(t *testing.T) {
	catalog := DefaultCatalog()

	// 3 minutes should return a 1-2 or 2-4 min exercise
	ex := Suggest(catalog, 3.0)
	if ex == nil {
		t.Fatal("Suggest returned nil")
	}
	if ex.MinMinutes > 3 {
		t.Errorf("exercise %q min=%d exceeds wait=3", ex.Name, ex.MinMinutes)
	}
}

func TestSuggestLongWait(t *testing.T) {
	catalog := DefaultCatalog()

	ex := Suggest(catalog, 10.0)
	if ex == nil {
		t.Fatal("Suggest returned nil")
	}
	if ex.MinMinutes > 10 {
		t.Errorf("exercise %q min=%d exceeds wait=10", ex.Name, ex.MinMinutes)
	}
}

func TestSuggestVeryShortWait(t *testing.T) {
	catalog := DefaultCatalog()

	// Even with a very short wait, should return something
	ex := Suggest(catalog, 0.5)
	if ex == nil {
		t.Fatal("Suggest returned nil for short wait")
	}
}

func TestSuggestVeryLongWait(t *testing.T) {
	catalog := DefaultCatalog()

	// Even with a very long wait, should return an exercise (all exercises fit)
	ex := Suggest(catalog, 60.0)
	if ex == nil {
		t.Fatal("Suggest returned nil for long wait")
	}
}

func TestSuggestCustomCatalog(t *testing.T) {
	catalog := []Exercise{
		{Name: "Quick stretch", Description: "Stretch", MinMinutes: 1, MaxMinutes: 2, Category: "stretch"},
	}

	ex := Suggest(catalog, 5.0)
	if ex == nil {
		t.Fatal("Suggest returned nil for custom catalog")
	}
	if ex.Name != "Quick stretch" {
		t.Errorf("Name = %q, want 'Quick stretch'", ex.Name)
	}
}

func TestSuggestSingleExerciseTooLong(t *testing.T) {
	catalog := []Exercise{
		{Name: "Long yoga", Description: "Yoga", MinMinutes: 10, MaxMinutes: 20, Category: "stretch"},
	}

	// Wait time is shorter than exercise min — should still return it via findShortest fallback
	ex := Suggest(catalog, 2.0)
	if ex == nil {
		t.Fatal("Suggest returned nil when only exercise is too long")
	}
	if ex.Name != "Long yoga" {
		t.Errorf("Name = %q, want 'Long yoga'", ex.Name)
	}
}

func TestSuggestEmptyCatalog(t *testing.T) {
	ex := Suggest(nil, 5.0)
	if ex != nil {
		t.Error("Suggest should return nil for empty catalog")
	}
}
