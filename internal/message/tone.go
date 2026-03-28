package message

import (
	"fmt"
	"math/rand"
)

// ExerciseSuggestion returns a varied message for an exercise suggestion during a build.
func ExerciseSuggestion(minutes float64, exerciseName, description string) string {
	intros := []string{
		fmt.Sprintf("This usually takes ~%.0fm.", minutes),
		fmt.Sprintf("~%.0fm to go.", minutes),
		fmt.Sprintf("Estimated %.0fm wait.", minutes),
		fmt.Sprintf("You've got ~%.0fm.", minutes),
	}

	bridges := []string{
		"Perfect time for:",
		"Why not try:",
		"Your body called. It wants:",
		"Your compiler doesn't care about your posture. We do. Try:",
		"Step away from the keyboard:",
		"Idle hands, meet:",
		"While you wait:",
	}

	intro := intros[rand.Intn(len(intros))]
	bridge := bridges[rand.Intn(len(bridges))]

	return fmt.Sprintf("buff-er: %s %s %s — %s", intro, bridge, exerciseName, description)
}

// BreakSuggestion returns a varied message for a time-since-break nudge.
func BreakSuggestion(elapsedMinutes int, exerciseName, description string) string {
	intros := []string{
		fmt.Sprintf("You've been at it for %dm without a break.", elapsedMinutes),
		fmt.Sprintf("%dm straight. Impressive, but your spine disagrees.", elapsedMinutes),
		fmt.Sprintf("%dm since your last break.", elapsedMinutes),
		fmt.Sprintf("It's been %dm. Your future self wants you to move.", elapsedMinutes),
		fmt.Sprintf("%dm of pure focus. Now move.", elapsedMinutes),
	}

	intro := intros[rand.Intn(len(intros))]
	return fmt.Sprintf("buff-er: %s Try: %s — %s", intro, exerciseName, description)
}

// FollowUp returns a varied message after a suggested command finishes.
func FollowUp(streak int) string {
	if streak > 0 {
		messages := []string{
			fmt.Sprintf("Done! Did you move? That's %d today.", streak),
			fmt.Sprintf("Back! %d exercise breaks today. Keep it up.", streak),
			fmt.Sprintf("Finished. You're at %d breaks today — not bad.", streak),
		}
		return "buff-er: " + messages[rand.Intn(len(messages))]
	}

	messages := []string{
		"Done! Did you get that exercise in? (you know the answer)",
		"Back! Did you move? Be honest with yourself.",
		"Finished! Hopefully you stretched. No judgment either way.",
		"Done! Your body thanks you. Or it would, if you moved.",
	}
	return "buff-er: " + messages[rand.Intn(len(messages))]
}
