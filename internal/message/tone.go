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

// BreakWarning returns a soft heads-up that a break is coming soon.
func BreakWarning() string {
	messages := []string{
		"You should take a break soon.",
		"Break incoming. Start wrapping up.",
		"Heads up — time to move soon.",
		"Your body is going to want a break shortly.",
		"Almost time to step away. Finish up your thought.",
	}
	return "buff-er: " + messages[rand.Intn(len(messages))]
}

// BreakNow returns the full exercise suggestion when the user kicks off a new task
// and should step away while it runs.
func BreakNow(exerciseName, description string) string {
	messages := []string{
		fmt.Sprintf("Things are in flight. Step away and do: %s — %s", exerciseName, description),
		fmt.Sprintf("AI's working, you should be stretching. Try: %s — %s", exerciseName, description),
		fmt.Sprintf("Let it run. Go do: %s — %s", exerciseName, description),
		fmt.Sprintf("Your tasks are running. Perfect time for: %s — %s", exerciseName, description),
	}
	return "buff-er: " + messages[rand.Intn(len(messages))]
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
