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

func TestSuggestEmptyCatalog(t *testing.T) {
	ex := Suggest(nil, 5.0)
	if ex != nil {
		t.Error("Suggest should return nil for empty catalog")
	}
}
