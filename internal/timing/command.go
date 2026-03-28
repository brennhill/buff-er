package timing

import (
	"strings"
	"unicode"
)

// ExtractPattern extracts a command pattern from a bash command string.
// It returns the first two tokens, normalized. For example:
//
//	"cargo build --release" -> "cargo build"
//	"npm test" -> "npm test"
//	"ENV=prod cargo build" -> "cargo build" (skips env vars)
//	"cd /foo && npm test" -> "npm test" (uses last command in chain)
//	"docker compose up -d" -> "docker compose"
func ExtractPattern(command string) string {
	// If there are pipes or chains, use the last command
	// (the long-running one is usually at the end)
	for _, sep := range []string{"&&", "||", "|", ";"} {
		if idx := strings.LastIndex(command, sep); idx >= 0 {
			command = command[idx+len(sep):]
		}
	}

	command = strings.TrimSpace(command)

	// Tokenize, skipping env var assignments (FOO=bar)
	tokens := tokenize(command)
	var filtered []string
	for _, t := range tokens {
		if strings.Contains(t, "=") && !strings.HasPrefix(t, "-") {
			continue // skip env var assignments
		}
		filtered = append(filtered, t)
		if len(filtered) == 2 {
			break
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	return strings.Join(filtered, " ")
}

// tokenize splits a command string into tokens, respecting quotes.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for _, r := range s {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case unicode.IsSpace(r) && !inSingle && !inDouble:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}
