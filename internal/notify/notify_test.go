package notify

import "testing"

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain text", "hello world", "hello world"},
		{"double quotes", `say "hello"`, `say \"hello\"`},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"newline replaced with space", "line1\nline2", "line1 line2"},
		{"carriage return replaced with space", "line1\rline2", "line1 line2"},
		{"crlf replaced with spaces", "line1\r\nline2", "line1  line2"},
		{
			"injection attempt via newline",
			"hello\"\ntell application \"Finder\" to delete",
			"hello\\\" tell application \\\"Finder\\\" to delete",
		},
		{"mixed escapes", `a"b\c` + "\n" + "d", `a\"b\\c d`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeAppleScript(tt.input)
			if got != tt.want {
				t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
