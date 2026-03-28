package message

import (
	"strings"
	"testing"
)

func TestExerciseSuggestion(t *testing.T) {
	msg := ExerciseSuggestion(5.0, "Desk pushups", "10 pushups against your desk edge")
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
	if !strings.Contains(msg, "Desk pushups") {
		t.Errorf("expected exercise name in message, got %q", msg)
	}
	if !strings.Contains(msg, "10 pushups against your desk edge") {
		t.Errorf("expected description in message, got %q", msg)
	}
}

func TestExerciseSuggestionZeroMinutes(t *testing.T) {
	msg := ExerciseSuggestion(0, "Stretch", "Quick stretch")
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
	// Should format as "0m" without panic
	if !strings.Contains(msg, "0m") {
		t.Errorf("expected '0m' in message, got %q", msg)
	}
}

func TestBreakSuggestion(t *testing.T) {
	msg := BreakSuggestion(45, "Walk around", "Get up and walk")
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
	if !strings.Contains(msg, "45") {
		t.Errorf("expected elapsed minutes in message, got %q", msg)
	}
	if !strings.Contains(msg, "Walk around") {
		t.Errorf("expected exercise name in message, got %q", msg)
	}
}

func TestFollowUpWithStreak(t *testing.T) {
	msg := FollowUp(3)
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
	if !strings.Contains(msg, "3") {
		t.Errorf("expected streak count in message, got %q", msg)
	}
}

func TestFollowUpHighStreak(t *testing.T) {
	msg := FollowUp(10)
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
	if !strings.Contains(msg, "10") {
		t.Errorf("expected streak count 10 in message, got %q", msg)
	}
}

func TestBreakSuggestionZeroMinutes(t *testing.T) {
	// Edge case: should not panic with zero elapsed minutes
	msg := BreakSuggestion(0, "Stretch", "Quick stretch")
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
}

func TestFollowUpZeroStreak(t *testing.T) {
	msg := FollowUp(0)
	if !strings.HasPrefix(msg, "buff-er: ") {
		t.Errorf("expected prefix 'buff-er: ', got %q", msg)
	}
	// With streak=0, should use the non-streak messages
	if strings.Contains(msg, " 0 ") {
		t.Errorf("zero streak should not show count, got %q", msg)
	}
}
